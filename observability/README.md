# Observability assets

Grafana dashboards and Prometheus alert rules for services built on colibri-sdk-go. They are
versioned next to the code that emits the metrics, and CI checks that every query they make
names a metric the SDK actually emits, so they cannot drift from the SDK.

```
observability/
  dashboards/
    http.json        REST server and REST client
    messaging.json   producers, consumers, consume lag
    database.json    SQL (sqlDB), Redis (cacheDB) and object storage
    runtime.json     Go runtime, GC and process
  alerts/
    rules.yaml       Prometheus alert rules
    rules_test.yaml  promtool unit tests of the rules
  check/             the metric catalog check run in CI
```

The metrics themselves, their attributes and the naming conventions are documented in
[docs/observability/metrics.md](../docs/observability/metrics.md).

## Metric names

The dashboards and rules query the names `/metrics` exposes, not the OpenTelemetry names: the
Prometheus reader of the SDK turns dots into underscores, appends the unit and adds `_total`
to counters, and attributes become labels the same way.

| OpenTelemetry                      | Prometheus                                    |
|------------------------------------|-----------------------------------------------|
| `http.server.request.duration`     | `http_server_request_duration_seconds_bucket` |
| `messaging.published`              | `messaging_published_total`                   |
| attribute `http.response.status_code` | label `http_response_status_code`          |

They work against any Prometheus-compatible backend scraping `/metrics` (Prometheus, Grafana
Mimir, VictoriaMetrics, Amazon Managed Prometheus, Grafana Cloud). A backend ingesting the
OTLP export instead may name the series differently and is not covered.

Every dashboard filters by the `job` and `instance` labels of the scrape. The `job` and
`instance` drop-downs are filled from `target_info`, which the SDK exposes on every service.

## Importing the dashboards

### Through the Grafana UI

1. **Dashboards → New → Import**, then upload one of the JSON files of `dashboards/`.
2. Save it. The dashboard has no datasource baked in: it uses the default Prometheus
   datasource of the Grafana instance until you pick another one.

### Through provisioning

Point a dashboard provider at a copy of `dashboards/`:

```yaml
# /etc/grafana/provisioning/dashboards/colibri.yaml
apiVersion: 1
providers:
  - name: colibri-sdk-go
    folder: Colibri
    type: file
    options:
      path: /var/lib/grafana/dashboards/colibri
```

[`development-environment/grafana`](../development-environment/grafana/provisioning) is a
working example.

### Choosing the datasource

Each dashboard has a **Datasource** drop-down, a variable of type `datasource` restricted to
Prometheus. Every panel queries `${datasource}`, so:

- in the UI, pick the datasource in the drop-down and save the dashboard to keep it as the
  default;
- with provisioning, either make the right datasource the default one (`isDefault: true`) or
  edit `templating.list[0].current` in the JSON to its name, e.g.
  `"current": {"text": "Mimir", "value": "mimir-uid"}`.

## Loading the alert rules

`alerts/rules.yaml` is a plain Prometheus rule file. Add it to `rule_files` in
`prometheus.yml`:

```yaml
rule_files:
  - /etc/prometheus/rules/rules.yaml
```

For the Prometheus Operator, wrap the `groups` in a `PrometheusRule` resource; for Grafana
Alerting or Mimir, import it with `mimirtool rules load` or `cortextool rules load`.

| Alert                              | Fires when                                                        | Severity |
|------------------------------------|-------------------------------------------------------------------|----------|
| `ColibriHTTPHighErrorRate`         | more than 5% of the requests answer 5xx for 5 minutes             | critical |
| `ColibriHTTPHighLatency`           | the p99 request duration is above 1s for 10 minutes               | warning  |
| `ColibriHTTPPanicRecovered`        | the REST server recovered a panic                                 | warning  |
| `ColibriMessagingPublishFailures`  | more than 1% of the publishes to a topic fail for 5 minutes       | critical |
| `ColibriMessagingConsumerBacklog`  | the p99 consume lag of a queue is above 60s for 10 minutes        | warning  |
| `ColibriMessagingConsumerPanics`   | a consumer panicked processing a message                          | warning  |
| `ColibriSQLPoolSaturated`          | queries wait for a connection with over 90% of the pool in use    | warning  |
| `ColibriGoMemoryNearLimit`         | the Go runtime uses more than 90% of `GOMEMLIMIT` for 10 minutes  | warning  |
| `ColibriGoGCPressure`              | more than 5% of the wall time goes to GC pauses for 10 minutes    | warning  |

The thresholds are starting points. Copy the file and tune them per service rather than
editing it in place, so SDK updates do not overwrite your changes.

The consumer backlog alert relies on `messaging.consume.lag`, the time a message waited in
the broker. The queue depth itself is reported by the broker (CloudWatch for SQS, Cloud
Monitoring for Pub/Sub, the RabbitMQ Prometheus plugin) and is not covered here.

## Trying them locally

The development environment runs Prometheus and Grafana with the dashboards and rules of
this directory mounted, so changes show up without an import:

```sh
cd development-environment
make observability   # only Prometheus and Grafana; `make start` brings up the emulators too
```

- Grafana: <http://localhost:3030>, anonymous admin, dashboards in the **Colibri** folder.
- Prometheus: <http://localhost:9091>, alerts under **Alerts**.

Prometheus scrapes `host.docker.internal:8080` and `:8081`, the school-module and
finantial-module of
[colibri-sdk-go-examples](https://github.com/colibriproject-dev/colibri-sdk-go-examples),
which exercise HTTP, messaging, SQL, cache and storage. Start them from that repository, or
change the targets in `development-environment/prometheus/prometheus.yml` to your own service
and reload with `curl -X POST localhost:9091/-/reload`.

## Checks

```sh
make observability        # both of the below
make observability-check  # every query names a metric of the SDK catalog
make observability-rules  # promtool check rules + promtool test rules (needs Docker)
```

The catalog check (`check/`) is a separate Go module, so the Prometheus server packages it
parses PromQL with never become a dependency of the SDK. It:

1. translates every entry of `monitoring.Catalog()` to the series `/metrics` exposes, with
   the same translator the Prometheus exporter of the SDK uses — a test records each metric
   through the real exporter to keep the two in step;
2. parses every panel target and variable query of the dashboards and every rule expression;
3. fails on a metric missing from the catalog, an expression that does not parse, or a
   selector without a metric name. `check/testdata/broken-dashboard.json` is a fixture that
   must fail.

A query may use Grafana's `$__rate_interval`, `$__interval` and `$__range`; other variables
belong inside label matchers (`job=~"$job"`).

### Changing a dashboard

Edit it in Grafana, export it with **Share → Export → Export for sharing externally** turned
**off**, so the `${datasource}` variable is kept, and replace the JSON file. Run
`make observability-check` before committing.

When a dashboard needs a metric the SDK does not emit yet, add the metric to the SDK and its
catalog first — see *Adding a metric to the SDK* in the
[metrics reference](../docs/observability/metrics.md#adding-a-metric-to-the-sdk).
