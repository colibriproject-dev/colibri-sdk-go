package colibri_otel

import (
	"context"
	"database/sql/driver"
	"slices"
	"strings"
	"unicode"

	"go.nhat.io/otelsql"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// otelsql (v0.16) emits pre-stable db conventions (db.statement, db.operation = driver
// method), but spans carry the stable db.system.name. Backends keyed on it (New Relic)
// then find no db.operation.name/db.collection.name and show "postgresql unknown", so
// the stable attributes are derived here from the SQL text.

const otelsqlScopeName = "go.nhat.io/otelsql"

// sqlQueryTracer records the SQL (without args) as db.query.text, plus db.namespace.
func sqlQueryTracer(dbName string) func(context.Context, string, []driver.NamedValue) []attribute.KeyValue {
	return func(_ context.Context, query string, _ []driver.NamedValue) []attribute.KeyValue {
		attrs := []attribute.KeyValue{semconv.DBQueryText(query)}
		if dbName != "" {
			attrs = append(attrs, semconv.DBNamespace(dbName))
		}

		return attrs
	}
}

// sqlSpanProcessor adds operation, collection and summary to otelsql spans. It runs on
// start because otelsql only exposes the SQL via the span's parent context.
type sqlSpanProcessor struct{}

var _ sdktrace.SpanProcessor = sqlSpanProcessor{}

func (sqlSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	if s.InstrumentationScope().Name != otelsqlScopeName {
		return
	}

	s.SetAttributes(sqlSpanAttributes(otelsql.QueryFromContext(parent), s.Name())...)
}

func (sqlSpanProcessor) OnEnd(sdktrace.ReadOnlySpan) {
	// Attributes are set on start; nothing to do once the span ends.
}

func (sqlSpanProcessor) Shutdown(context.Context) error   { return nil }
func (sqlSpanProcessor) ForceFlush(context.Context) error { return nil }

// sqlSpanAttributes describes a span from its SQL. Spans without SQL (commit, rows_close…)
// fall back to the method from the "sql:<method>" span name.
func sqlSpanAttributes(query, spanName string) []attribute.KeyValue {
	operation, collection := parseSQL(query)
	if operation == "" {
		method := strings.TrimPrefix(spanName, "sql:")
		if method == "" {
			return nil
		}

		if method == "begin_transaction" {
			method = "begin"
		}

		return []attribute.KeyValue{semconv.DBOperationName(strings.ToUpper(method))}
	}

	attrs := make([]attribute.KeyValue, 0, 3)
	attrs = append(attrs, semconv.DBOperationName(operation))

	summary := operation
	if collection != "" {
		attrs = append(attrs, semconv.DBCollectionName(collection))
		summary += " " + collection
	}

	return append(attrs, semconv.DBQuerySummary(summary))
}

// sqlMainStatements are the statements a WITH clause can lead into.
var sqlMainStatements = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "MERGE"}

// parseSQL returns the leading keyword (upper-cased) and the first table the statement
// acts on. Not a full parser: only tokens at the statement's parenthesis depth count.
func parseSQL(query string) (operation, collection string) {
	tokens := tokenizeSQL(query)

	start := slices.IndexFunc(tokens, func(t sqlToken) bool { return t.word })
	if start < 0 {
		return "", ""
	}

	operation = strings.ToUpper(tokens[start].text)
	depth := tokens[start].depth

	// CTE bodies are nested deeper; the main statement is the next one at the WITH level.
	if operation == "WITH" {
		i := slices.IndexFunc(tokens[start+1:], func(t sqlToken) bool {
			return t.word && t.depth == depth && slices.Contains(sqlMainStatements, strings.ToUpper(t.text))
		})
		if i < 0 {
			return operation, ""
		}

		start += 1 + i
		operation = strings.ToUpper(tokens[start].text)
	}

	rest := tokens[start+1:]

	switch operation {
	case "SELECT", "DELETE":
		return operation, tableAfter(rest, depth, "FROM")
	case "INSERT", "MERGE":
		return operation, tableAfter(rest, depth, "INTO")
	case "UPDATE", "CALL":
		return operation, tableAt(rest, depth)
	case "TRUNCATE":
		if len(rest) > 0 && strings.EqualFold(rest[0].text, "TABLE") {
			rest = rest[1:]
		}

		return operation, tableAt(rest, depth)
	default:
		return operation, ""
	}
}

// tableAfter returns the table following the first keyword found at the given depth.
func tableAfter(tokens []sqlToken, depth int, keyword string) string {
	i := slices.IndexFunc(tokens, func(t sqlToken) bool {
		return t.word && t.depth == depth && strings.EqualFold(t.text, keyword)
	})
	if i < 0 {
		return ""
	}

	return tableAt(tokens[i+1:], depth)
}

// tableAt reads a possibly schema-qualified table name at the head of tokens, skipping
// ONLY. A subquery in table position, "FROM (SELECT …)", yields no name.
func tableAt(tokens []sqlToken, depth int) string {
	if len(tokens) > 0 && tokens[0].word && strings.EqualFold(tokens[0].text, "ONLY") {
		tokens = tokens[1:]
	}

	if len(tokens) == 0 || !tokens[0].ident() || tokens[0].depth != depth {
		return ""
	}

	parts := []string{tokens[0].text}
	for i := 1; i+1 < len(tokens) && tokens[i].text == "." && tokens[i+1].ident(); i += 2 {
		parts = append(parts, tokens[i+1].text)
	}

	return strings.Join(parts, ".")
}

// sqlToken is a word, a quoted identifier (unquoted in text) or a single punctuation
// character, together with the parenthesis depth it was found at.
type sqlToken struct {
	text   string
	depth  int
	word   bool
	quoted bool
}

func (t sqlToken) ident() bool { return t.word || t.quoted }

// tokenizeSQL splits a query into tokens, dropping whitespace, comments, string literals
// (including PostgreSQL dollar-quoted ones), numbers and placeholders.
func tokenizeSQL(query string) []sqlToken {
	var (
		tokens []sqlToken
		depth  int
	)

	for i := 0; i < len(query); {
		c := query[i]

		switch {
		case c == '-' && strings.HasPrefix(query[i:], "--"):
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				return tokens
			}
			i += end + 1

		case c == '/' && strings.HasPrefix(query[i:], "/*"):
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return tokens
			}
			i += 2 + end + 2

		case c == '\'':
			i = skipQuoted(query, i, '\'')

		case c == '"':
			end := skipQuoted(query, i, '"')
			name := strings.ReplaceAll(query[i+1:max(i+1, end-1)], `""`, `"`)
			tokens = append(tokens, sqlToken{text: name, depth: depth, quoted: true})
			i = end

		case c == '$':
			i = skipDollar(query, i)

		case c == '(':
			tokens = append(tokens, sqlToken{text: "(", depth: depth})
			depth++
			i++

		case c == ')':
			depth = max(depth-1, 0)
			tokens = append(tokens, sqlToken{text: ")", depth: depth})
			i++

		case isWordStart(c):
			end := i + 1
			for end < len(query) && isWordPart(query[end]) {
				end++
			}
			tokens = append(tokens, sqlToken{text: query[i:end], depth: depth, word: true})
			i = end

		case c >= '0' && c <= '9':
			for i < len(query) && (isWordPart(query[i]) || query[i] == '.') {
				i++
			}

		case unicode.IsSpace(rune(c)):
			i++

		default:
			tokens = append(tokens, sqlToken{text: string(c), depth: depth})
			i++
		}
	}

	return tokens
}

// skipQuoted returns the index right after the literal opened at query[start], where a
// doubled quote is an escaped one. An unterminated literal runs to the end.
func skipQuoted(query string, start int, quote byte) int {
	for i := start + 1; i < len(query); i++ {
		if query[i] != quote {
			continue
		}

		if i+1 < len(query) && query[i+1] == quote {
			i++
			continue
		}

		return i + 1
	}

	return len(query)
}

// skipDollar handles the '$' at query[start]: a placeholder ($1) or a dollar-quoted
// string ($$…$$, $tag$…$tag$). It returns the index right after it.
func skipDollar(query string, start int) int {
	i := start + 1
	if i < len(query) && query[i] >= '0' && query[i] <= '9' {
		for i < len(query) && query[i] >= '0' && query[i] <= '9' {
			i++
		}

		return i
	}

	for i < len(query) && query[i] != '$' && isWordPart(query[i]) {
		i++
	}

	if i >= len(query) || query[i] != '$' {
		return i // a lone '$', not a tag
	}

	tag := query[start : i+1]
	if end := strings.Index(query[i+1:], tag); end >= 0 {
		return i + 1 + end + len(tag)
	}

	return len(query)
}

func isWordStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isWordPart(c byte) bool {
	return isWordStart(c) || (c >= '0' && c <= '9') || c == '$'
}
