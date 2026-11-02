package colibri_otel

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestParseSQL(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		operation  string
		collection string
	}{
		{"select", "SELECT id, name FROM plans WHERE active = $1", "SELECT", "plans"},
		{"lower case", "select * from plans", "SELECT", "plans"},
		{"schema qualified", "SELECT * FROM public.plans", "SELECT", "public.plans"},
		{"quoted identifiers", `SELECT * FROM "public"."Plan ""X"""`, "SELECT", `public.Plan "X"`},
		{"join keeps the first table", "SELECT p.id FROM plans p JOIN company c ON c.id = p.company_id", "SELECT", "plans"},
		{"no from", "SELECT 1", "SELECT", ""},
		{"from inside a function is ignored", "SELECT EXTRACT(year FROM created_at) FROM rental", "SELECT", "rental"},
		{"subquery in select list", "SELECT (SELECT count(*) FROM item i WHERE i.r = r.id) FROM rental r", "SELECT", "rental"},
		{"subquery in from", "SELECT * FROM (SELECT * FROM plans) p", "SELECT", ""},
		{"leading comments and whitespace", "-- lista\n /* planos */\n\tSELECT * FROM plans", "SELECT", "plans"},
		{"keywords inside literals are ignored", "SELECT 'x FROM y', $$ FROM z $$, $t$ FROM w $t$ FROM plans", "SELECT", "plans"},
		{"escaped quote in literal", "SELECT 'it''s FROM x' FROM plans", "SELECT", "plans"},
		{"insert", "INSERT INTO legal_acceptance (id, version) VALUES ($1, $2) RETURNING id", "INSERT", "legal_acceptance"},
		{"update", "UPDATE rental SET status = $1 WHERE id = $2", "UPDATE", "rental"},
		{"update only", "UPDATE ONLY rental SET status = 'x'", "UPDATE", "rental"},
		{"delete", "DELETE FROM expense WHERE id = $1", "DELETE", "expense"},
		{"cte", "WITH recent AS (SELECT * FROM rental WHERE x > 1) SELECT * FROM recent", "SELECT", "recent"},
		{"cte into insert", "WITH s AS (SELECT 1) INSERT INTO audit (v) SELECT * FROM s", "INSERT", "audit"},
		{"merge", "MERGE INTO stock s USING incoming i ON s.id = i.id WHEN MATCHED THEN DELETE", "MERGE", "stock"},
		{"call", "CALL refresh_stats($1)", "CALL", "refresh_stats"},
		{"truncate", "TRUNCATE TABLE nfse_outbox", "TRUNCATE", "nfse_outbox"},
		{"ddl has no table", "CREATE INDEX idx ON rental (company_id)", "CREATE", ""},
		{"parenthesised select", "(SELECT * FROM a) UNION (SELECT * FROM b)", "SELECT", "a"},
		{"empty", "", "", ""},
		{"only a comment", "-- nada", "", ""},
		{"unterminated literal", "SELECT 'abc", "SELECT", ""},
		{"unterminated quoted identifier", `SELECT * FROM "plans`, "SELECT", "plans"},
		{"unterminated block comment", "SELECT * FROM plans /* x", "SELECT", "plans"},
		{"numbers are skipped", "SELECT 1.5e3 FROM plans", "SELECT", "plans"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation, collection := parseSQL(tt.query)

			assert.Equal(t, tt.operation, operation)
			assert.Equal(t, tt.collection, collection)
		})
	}
}

func TestSQLSpanAttributes(t *testing.T) {
	t.Run("operation, collection and summary from the SQL", func(t *testing.T) {
		attrs := sqlSpanAttributes("SELECT * FROM plans", "sql:query")

		assert.ElementsMatch(t, []attribute.KeyValue{
			semconv.DBOperationName("SELECT"),
			semconv.DBCollectionName("plans"),
			semconv.DBQuerySummary("SELECT plans"),
		}, attrs)
	})

	t.Run("summary is the operation alone without a table", func(t *testing.T) {
		attrs := sqlSpanAttributes("SELECT 1", "sql:query")

		assert.ElementsMatch(t, []attribute.KeyValue{
			semconv.DBOperationName("SELECT"),
			semconv.DBQuerySummary("SELECT"),
		}, attrs)
	})

	t.Run("method from the span name without SQL", func(t *testing.T) {
		attrs := sqlSpanAttributes("", "sql:commit")

		assert.Equal(t, []attribute.KeyValue{semconv.DBOperationName("COMMIT")}, attrs)
	})
}

// TestSQLDriverSpans runs the SDK's otelsql options over a fake driver and checks every
// span carries the stable attributes a backend needs to name the database segment.
func TestSQLDriverSpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sqlSpanProcessor{}),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	sql.Register("colibri-fake-sql", fakeDriver{})
	driverName, err := otelsql.Register("colibri-fake-sql",
		append(sqlDriverOptions("localoop"), otelsql.WithTracerProvider(tp))...,
	)
	require.NoError(t, err)

	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := t.Context()

	rows, err := db.QueryContext(ctx, "SELECT id FROM plans WHERE active = $1", true)
	require.NoError(t, err)
	require.NoError(t, rows.Close())

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	result, err := tx.ExecContext(ctx, "UPDATE rental SET status = $1", "x")
	require.NoError(t, err)
	_, err = result.RowsAffected()
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	spans := map[string]map[attribute.Key]attribute.Value{}
	for _, s := range recorder.Ended() {
		attrs := map[attribute.Key]attribute.Value{}
		for _, kv := range s.Attributes() {
			attrs[kv.Key] = kv.Value
		}
		spans[s.Name()+" "+attrs[semconv.DBQuerySummaryKey].AsString()] = attrs
	}

	query := spans["sql:query SELECT plans"]
	require.NotNil(t, query, "spans: %v", slices.Collect(maps.Keys(spans)))
	assert.Equal(t, "SELECT id FROM plans WHERE active = $1", query[semconv.DBQueryTextKey].AsString())
	assert.Equal(t, "localoop", query[semconv.DBNamespaceKey].AsString())
	assert.Equal(t, "postgresql", query[semconv.DBSystemNameKey].AsString())
	assert.NotContains(t, query, attribute.Key("db.statement"))

	assert.Contains(t, spans, "sql:exec UPDATE rental")

	// No SQL in context: operation comes from the span name.
	for span, operation := range map[string]string{
		"sql:rows_close ":        "ROWS_CLOSE",
		"sql:rows_affected ":     "ROWS_AFFECTED",
		"sql:begin_transaction ": "BEGIN",
		"sql:commit ":            "COMMIT",
	} {
		require.Contains(t, spans, span)
		assert.Equal(t, operation, spans[span][semconv.DBOperationNameKey].AsString(), span)
	}

	for name, attrs := range spans {
		assert.Contains(t, attrs, semconv.DBOperationNameKey, "span %q has no db.operation.name", name)
	}
}

// fakeDriver is the least of a database/sql driver otelsql needs to wrap: queries return
// no rows and every statement succeeds.
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return fakeTx{}, nil }

func (fakeConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return fakeRows{}, nil
}

func (fakeConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

type fakeRows struct{}

func (fakeRows) Columns() []string         { return []string{"id"} }
func (fakeRows) Close() error              { return nil }
func (fakeRows) Next([]driver.Value) error { return io.EOF }
