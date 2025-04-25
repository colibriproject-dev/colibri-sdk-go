package colibri_nr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	colibri_monitoring_base "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"

	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
	"github.com/newrelic/go-agent/v3/newrelic"
)

const pgDriverName = "nrpostgres"

type MonitoringNewRelic struct {
	*newrelic.Application
}

func StartNewRelicMonitoring() colibri_monitoring_base.Monitoring {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(config.APP_NAME),
		newrelic.ConfigLicense(config.NEW_RELIC_LICENSE),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		logging.
			Fatal(context.Background()).
			Err(err).
			Msg("an error occurred while loading the monitoring provider")
	}

	return &MonitoringNewRelic{app}
}

func (m *MonitoringNewRelic) StartTransaction(ctx context.Context, name string) (interface{}, context.Context) {
	transaction := m.Application.StartTransaction(name)
	ctx = newrelic.NewContext(ctx, transaction)

	return transaction, ctx
}

func (m *MonitoringNewRelic) EndTransaction(transaction interface{}) {
	transaction.(*newrelic.Transaction).End()
}

func (m *MonitoringNewRelic) setWebRequest(transaction interface{}, header http.Header, url *url.URL, method string) {
	transaction.(*newrelic.Transaction).SetWebRequest(newrelic.WebRequest{
		Header:    header,
		URL:       url,
		Method:    method,
		Transport: newrelic.TransportHTTP,
	})
}

func (m *MonitoringNewRelic) StartWebRequest(ctx context.Context, header http.Header, path string, method string) (interface{}, context.Context) {
	txn, ctx := m.StartTransaction(ctx, fmt.Sprintf("%s %s", method, path))
	m.setWebRequest(txn, header, &url.URL{Path: path}, method)

	return txn, ctx
}

func (m *MonitoringNewRelic) SetWebResponse(transaction interface{}, w http.ResponseWriter) http.ResponseWriter {
	return transaction.(*newrelic.Transaction).SetWebResponse(w)
}

func (m *MonitoringNewRelic) StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) interface{} {
	transaction := m.GetTransactionInContext(ctx)
	segment := transaction.(*newrelic.Transaction).StartSegment(name)

	for key, value := range attributes {
		segment.AddAttribute(key, value)
	}

	return segment
}

func (m *MonitoringNewRelic) EndTransactionSegment(segment interface{}) {
	segment.(*newrelic.Segment).End()
}

func (m *MonitoringNewRelic) GetTransactionInContext(ctx context.Context) interface{} {
	return newrelic.FromContext(ctx)
}

func (m *MonitoringNewRelic) NoticeError(transaction interface{}, err error) {
	transaction.(*newrelic.Transaction).NoticeError(err)
}

func (m *MonitoringNewRelic) GetSQLDBDriverName() string {
	return pgDriverName
}
