# Metrics

This is the reference for the metrics the SDK emits and the conventions every metric —
the SDK's and the application's — follows. How to enable metrics, scrape `/metrics` or
export over OTLP is covered in the [README](../../README.md#observability).

- [Naming](#naming)
- [Units](#units)
- [Cardinality](#cardinality)
- [Business metrics](#business-metrics)
- [Catalog](#catalog)

## Naming

- **Reuse the OpenTelemetry semantic conventions where they define the metric.** The HTTP
  server metrics are `http.server.request.duration` and friends, not a name of our own:
  backends such as New Relic build their views from the semconv names.
- **Otherwise use plain domain naming**: `<domain>.<thing>[.<qualifier>]`, lowercase,
  dot-separated, with `snake_case` inside a segment — `messaging.process.duration`,
  `storage.operation`, `messaging.in_flight`.
- **No `colibri.*` prefix.** Whether a metric comes from the SDK or from a third-party
  instrumentation is recorded in the [catalog](#catalog), not in the name.
- **Leave the unit and the type out of the name.** No `_seconds`, `_bytes`, `_total` or
  `_count`: the unit goes in the unit field, and the Prometheus exporter appends the
  suffixes itself.
- **Name what is measured, not the outcome.** Split success and failure with an attribute
  (`result`), not with two metrics.

### Names on `/metrics`

The Prometheus reader translates the OpenTelemetry name: dots become underscores, the unit
is appended as a suffix and counters get `_total`. Dashboards and alert rules query the
translated name.

| OpenTelemetry name                       | Unit        | Prometheus name                      |
|------------------------------------------|-------------|--------------------------------------|
| `messaging.published` (counter)          | `{message}` | `messaging_published_total`          |
| `messaging.process.duration` (histogram) | `s`         | `messaging_process_duration_seconds` |
| `storage.transferred` (histogram)        | `By`        | `storage_transferred_bytes`          |
| `messaging.in_flight` (gauge)            | `{message}` | `messaging_in_flight`                |
| `db.sql.connections.open` (gauge)        | `1`         | `db_sql_connections_open_ratio`      |
| `db.sql.client.latency` (histogram)      | `ms`        | `db_sql_client_latency_milliseconds` |

Annotations such as `{message}` produce no suffix, and histograms are exposed as the
`_bucket`, `_sum` and `_count` series of the translated name.

`/metrics` also serves the Go and process collectors of the Prometheus client
(`go_gc_duration_seconds`, `go_memstats_*`, `process_*`). They are named by the Prometheus
client itself, are not translated and never reach the OTLP exporter. The catalog lists them
under the `promclient` origin, so dashboards and alert rules can be checked against them too.

## Units

Units follow [UCUM](https://ucum.org/ucum), as the OpenTelemetry conventions require.

| Measuring              | Unit                             | Example                       |
|------------------------|----------------------------------|-------------------------------|
| Durations              | `s`, as a float                  | `messaging.process.duration`  |
| Sizes                  | `By`                             | `storage.transferred`         |
| Counts of things       | an annotation in braces          | `{message}`, `{request}`      |
| Dimensionless ratios   | `1`                              | a cache hit ratio             |
| Percentages            | `%`                              | `go.config.gogc`              |

Record durations in seconds, never milliseconds: it is what the semantic conventions and
the Prometheus ecosystem expect. `ms`, `ns` and missing units appear in the catalog only in
metrics from third-party instrumentation, which the SDK does not control.

## Cardinality

Every distinct combination of attribute values is a separate time series, stored and
billed on its own. An attribute taking unbounded values turns one metric into millions of
series, so **every attribute must come from a small, fixed set of values**.

These are never metric attributes. They belong on spans, where each request is a record
anyway:

- request identifiers: `correlationId`, `messageId`, trace and span ids;
- entity identifiers: `userId`, `tenantId`, order, payment or document ids, e-mails;
- storage object keys and file names;
- raw URL paths and query strings — use the route template (`/users/:id`), never the
  path (`/users/42`);
- error messages, free text and timestamps.

Also keep in mind:

- **At most 5 attributes per metric**, fewer where possible. The catalog test enforces the
  ceiling on the SDK metrics.
- **Keep a metric under about a thousand series per instance.** Multiply the number of
  values of each attribute to estimate it.
- **Values set by the application count too.** The `action` of a message is set on
  `Publish`, so it must be an event name from a fixed set, never an identifier.
- **Outcomes use `result`**, with `success` and `error` (and `panic` where it applies).

### Known exceptions

Two metrics from outside the SDK's control carry attributes that bend this policy:

- `db.sql.client.latency` and `db.sql.client.calls` (otelsql) carry `db.sql.error`, the
  message of a failed call. Each distinct error message becomes a series.
- the `http.server.*` metrics carry `server.address`, as the semantic conventions specify.
  It is read from the `Host` header, which the client controls: behind an ingress or load
  balancer that only routes known hosts it stays bounded, but a service exposed directly
  can be sent any value.

## Business metrics

Application metrics go through the same API the SDK uses. Create the instruments once, at
setup, and keep them:

```go
import (
    "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
    monitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
)

type paymentMetrics struct {
    authorized monitoringbase.Counter
    latency    monitoringbase.HistogramRecorder
}

// newPaymentMetrics runs after colibri.InitializeApp, e.g. from the service constructor.
func newPaymentMetrics() *paymentMetrics {
    return &paymentMetrics{
        authorized: monitoring.Counter("payments.authorized", "Number of payment authorizations", "{payment}"),
        latency: monitoring.Histogram("payments.authorization.duration",
            "Duration of payment authorizations", "s"),
    }
}
```

> **Create instruments after `colibri.InitializeApp()`.** Before monitoring is initialized
> the accessors return noop instruments, so a package-level
> `var c = monitoring.Counter(...)` records nothing, ever. With metrics disabled the
> instruments are noop as well: code never needs to check.

Name business metrics after the domain — `payments.authorized`,
`payments.authorization.duration` — following the [naming](#naming) rules.

### Attributes

Attributes are passed as an `Attrs` value. It caches its provider representation, so
**build it once and reuse it**: recording with a reused `Attrs` does not allocate.

```go
// Built once, outside the hot path: one value per combination.
var (
    cardApproved = monitoringbase.NewAttrs("method", "card", "result", "success")
    cardDeclined = monitoringbase.NewAttrs("method", "card", "result", "error")
)

func (s *PaymentService) authorize(ctx context.Context, order Order) error {
    start := time.Now()
    err := s.gateway.Authorize(ctx, order)

    attrs := cardApproved
    if err != nil {
        attrs = cardDeclined
    }
    s.metrics.authorized.AddAttrs(ctx, 1, attrs)
    s.metrics.latency.RecordAttrs(ctx, time.Since(start).Seconds(), attrs)
    return err
}
```

When the values are only known at runtime but come from a bounded set, build each set
once and keep it in a map, the way the messaging module caches one per `action`.

`Add` and `Record` taking a `map[string]string` still work and are deprecated: they convert
the map on every call.

```go
// Before
requests.Add(ctx, 1, map[string]string{"route": "/api/users"})

// After — build once, reuse
var usersRoute = monitoringbase.NewAttrs("route", "/api/users")
requests.AddAttrs(ctx, 1, usersRoute)

// Or, to migrate mechanically from an existing map
requests.AddAttrs(ctx, 1, monitoringbase.AttrsFromMap(attributes))
```

### Gauges

`Gauge` records a value the application pushes. For a value that is sampled rather than
pushed — pool size, queue depth, cache entries — register an observable gauge, whose
callback runs on every collection:

```go
registration := monitoring.ObservableGauge(
    "orders.pending", "Number of orders waiting for payment", "{order}",
    func(ctx context.Context) []monitoringbase.Observation {
        return []monitoringbase.Observation{
            {Value: float64(queue.Len()), Attributes: monitoringbase.NewAttrs("priority", "high")},
        }
    },
)
defer registration.Unregister()
```

Register a given name once: each call registers its own callback, and the caller owns its
lifetime through the returned `Registration`.

### Testing

`monitoringtest.Install(t)` swaps in an in-memory reader for the duration of the test, and
installs it as the global meter provider too, so third-party instrumentation records into
it. Install it before creating the instruments under test.

```go
recorder := monitoringtest.Install(t)
// ... exercise the code ...
created := recorder.Metric(t, "orders.created")
monitoringtest.AssertShape(t, created, "{order}", "channel")
assert.Equal(t, int64(1), monitoringtest.CounterValue(t, created, "channel", "web"))
```

`AssertShape` checks the unit and that every data point carries exactly the given
attribute keys, which keeps an unexpected attribute from slipping in.

## Catalog

Every metric the SDK emits is listed in
[`pkg/base/monitoring/metrics_catalog.go`](../../pkg/base/monitoring/metrics_catalog.go),
with its name as an exported constant (`monitoring.MetricMessagingPublished`), its type,
unit, attribute keys and origin. `monitoring.Catalog()` and `monitoring.LookupMetric` expose
it to tooling, such as checks that dashboards only query metrics that exist.

- **Origin** is who emits the metric: `sdk` for the SDK's own code, or the third-party
  instrumentation it wires (`otelhttp`, `otelsql`, `redisotel`, `runtime`). `promclient`
  marks the Prometheus client collectors, whose name is already the Prometheus one.
- **Module** is the SDK package whose setup enables the metric.
- **Attributes** are the keys a metric may carry. An SDK metric carries all of them; a
  third-party one may leave some out, such as the status code of a request that failed
  before a response.

The tables at the end of this section are generated from the catalog; do not edit them by hand.

### Attribute values

- `result` is `success` or `error`; consumed messages also use `panic`. `reason` is
  `error` or `panic`.
- `action` is the event name the application sets on `Publish`.
- `operation` is `upload`, `download` or `delete`.
- `http.route` is the route template. A panic raised in a middleware, before the route
  handler sets the template, is recorded with the matched route, `/` for a middleware.
- `messaging.consume.lag` is measured from the publish time the broker reports (the SNS
  notification timestamp, or the SQS `SentTimestamp`; the Pub/Sub publish time; the AMQP
  `timestamp` property, which the SDK producer sets) to the moment the consumer picks the
  message up. A message without a publish time is not recorded, and clock skew that puts the
  publish time ahead of the consumer is recorded as zero. It is the SDK signal for a growing
  backlog; the queue depth itself is reported by the broker.
- `messaging.rejected` counts messages nacked without requeue. The SDK leaves them to the
  broker dead-letter handling (SQS redrive policy, Pub/Sub dead-letter topic, RabbitMQ DLX),
  so whether one actually reached a DLQ is reported by the broker, not by the SDK.

### Adding a metric to the SDK

1. Add the name constant and the entry to `metrics_catalog.go`.
2. Create the instrument with the catalog constant and the same description and unit.
3. Assert on it in the module tests with `monitoringtest.AssertCataloged`, which checks the
   metric against its catalog entry.
4. Run `make metrics-doc` to regenerate the tables. A test fails while they are out of
   date.

<!-- metrics:begin -->

#### Emitted by the SDK

| Metric | Type | Unit | Attributes | Module | Origin | Description |
|--------|------|------|------------|--------|--------|-------------|
| `http.server.request.duration` | histogram | `s` | `http.request.method`, `http.response.status_code`, `http.route`, `server.address`, `url.scheme` | restserver | sdk | Duration of HTTP server requests |
| `http.server.active_requests` | updowncounter | `{request}` | `http.request.method`, `server.address`, `url.scheme` | restserver | sdk | Number of active HTTP server requests |
| `http.server.request.body.size` | histogram | `By` | `http.request.method`, `http.response.status_code`, `http.route`, `server.address`, `url.scheme` | restserver | sdk | Size of HTTP server request bodies |
| `http.server.response.body.size` | histogram | `By` | `http.request.method`, `http.response.status_code`, `http.route`, `server.address`, `url.scheme` | restserver | sdk | Size of HTTP server response bodies |
| `http.server.panic.recovered` | counter | `{panic}` | `http.request.method`, `http.route` | restserver | sdk | Number of panics recovered while serving HTTP requests |
| `messaging.published` | counter | `{message}` | `topic`, `result` | messaging | sdk | Number of messages published |
| `messaging.consumed` | counter | `{message}` | `queue`, `action`, `result` | messaging | sdk | Number of messages consumed |
| `messaging.process.duration` | histogram | `s` | `queue`, `action`, `result` | messaging | sdk | Duration of the processing of a consumed message |
| `messaging.rejected` | counter | `{message}` | `queue`, `action`, `reason` | messaging | sdk | Number of consumed messages rejected without requeue, left to the broker dead-letter handling |
| `messaging.in_flight` | observable_gauge | `{message}` | `queue` | messaging | sdk | Number of messages being processed |
| `messaging.consume.lag` | histogram | `s` | `queue`, `action` | messaging | sdk | Time between the broker accepting a message and a consumer receiving it |
| `storage.operation` | counter | `{operation}` | `operation`, `result` | storage | sdk | Number of storage operations |
| `storage.operation.duration` | histogram | `s` | `operation`, `result` | storage | sdk | Duration of storage operations |
| `storage.transferred` | histogram | `By` | `operation` | storage | sdk | Size of the files uploaded to and downloaded from the storage |

#### Emitted by third-party instrumentation

| Metric | Type | Unit | Attributes | Module | Origin | Description |
|--------|------|------|------------|--------|--------|-------------|
| `http.client.request.duration` | histogram | `s` | `http.request.method`, `http.response.status_code`, `network.protocol.name`, `network.protocol.version`, `server.address`, `server.port`, `url.scheme` | restclient | otelhttp | Duration of HTTP client requests. |
| `http.client.request.body.size` | histogram | `By` | `http.request.method`, `http.response.status_code`, `network.protocol.name`, `network.protocol.version`, `server.address`, `server.port`, `url.scheme` | restclient | otelhttp | Size of HTTP client request bodies. |
| `db.sql.client.latency` | histogram | `ms` | `db.name`, `db.operation`, `db.sql.error`, `db.sql.status`, `db.system.name` | sqlDB | otelsql | The distribution of latencies of various calls in milliseconds |
| `db.sql.client.calls` | counter | `1` | `db.name`, `db.operation`, `db.sql.error`, `db.sql.status`, `db.system.name` | sqlDB | otelsql | The number of various calls of methods |
| `db.sql.connections.open` | observable_gauge | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | Count of open connections in the pool |
| `db.sql.connections.idle` | observable_gauge | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | Count of idle connections in the pool |
| `db.sql.connections.active` | observable_gauge | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | Count of active connections in the pool |
| `db.sql.connections.wait_count` | observable_counter | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | The total number of connections waited for |
| `db.sql.connections.wait_duration` | observable_counter | `ms` | `db.instance`, `db.system.name` | sqlDB | otelsql | The total time blocked waiting for a new connection |
| `db.sql.connections.idle_closed` | observable_counter | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | The total number of connections closed due to SetMaxIdleConns |
| `db.sql.connections.idle_time_closed` | observable_counter | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | The total number of connections closed due to SetConnMaxIdleTime |
| `db.sql.connections.lifetime_closed` | observable_counter | `1` | `db.instance`, `db.system.name` | sqlDB | otelsql | The total number of connections closed due to SetConnMaxLifetime |
| `db.client.connections.idle.max` | observable_updowncounter | — | `db.system`, `pool.name` | cacheDB | redisotel | The maximum number of idle open connections allowed |
| `db.client.connections.idle.min` | observable_updowncounter | — | `db.system`, `pool.name` | cacheDB | redisotel | The minimum number of idle open connections allowed |
| `db.client.connections.max` | observable_updowncounter | — | `db.system`, `pool.name` | cacheDB | redisotel | The maximum number of open connections allowed |
| `db.client.connections.usage` | observable_updowncounter | — | `db.system`, `pool.name`, `state` | cacheDB | redisotel | The number of connections that are currently in state described by the state attribute |
| `db.client.connections.waits` | observable_counter | — | `db.system`, `pool.name` | cacheDB | redisotel | The number of times a connection was waited for |
| `db.client.connections.waits_duration` | observable_updowncounter | `ns` | `db.system`, `pool.name` | cacheDB | redisotel | The total time spent for waiting a connection in nanoseconds |
| `db.client.connections.timeouts` | observable_counter | — | `db.system`, `pool.name` | cacheDB | redisotel | The number of connection timeouts that have occurred trying to obtain a connection from the pool |
| `db.client.connections.hits` | observable_counter | — | `db.system`, `pool.name` | cacheDB | redisotel | The number of times free connection was found in the pool |
| `db.client.connections.misses` | observable_counter | — | `db.system`, `pool.name` | cacheDB | redisotel | The number of times free connection was not found in the pool |
| `db.client.connections.create_time` | histogram | `ms` | `db.system`, `error_type`, `pool.name`, `status` | cacheDB | redisotel | The time it took to create a new connection. |
| `db.client.connections.use_time` | histogram | `ms` | `db.system`, `error_type`, `pool.name`, `status`, `type` | cacheDB | redisotel | The time between borrowing a connection and returning it to the pool. |
| `go.memory.used` | observable_updowncounter | `By` | `go.memory.type` | monitoring | runtime | Memory used by the Go runtime. |
| `go.memory.limit` | observable_updowncounter | `By` | — | monitoring | runtime | Go runtime memory limit configured by the user, if a limit exists. |
| `go.memory.allocated` | observable_counter | `By` | — | monitoring | runtime | Memory allocated to the heap by the application. |
| `go.memory.allocations` | observable_counter | `{allocation}` | — | monitoring | runtime | Count of allocations to the heap by the application. |
| `go.memory.gc.goal` | observable_updowncounter | `By` | — | monitoring | runtime | Heap size target for the end of the GC cycle. |
| `go.goroutine.count` | observable_updowncounter | `{goroutine}` | — | monitoring | runtime | Count of live goroutines. |
| `go.processor.limit` | observable_updowncounter | `{thread}` | — | monitoring | runtime | The number of OS threads that can execute user-level Go code simultaneously. |
| `go.config.gogc` | observable_updowncounter | `%` | — | monitoring | runtime | Heap size target percentage configured by the user, otherwise 100. |

#### Exposed only on `/metrics`, by the Prometheus client collectors

| Metric | Type | Unit | Attributes | Module | Origin | Description |
|--------|------|------|------------|--------|--------|-------------|
| `go_gc_duration_seconds` | summary | `s` | `quantile` | restserver | promclient | A summary of the wall-time pause (stop-the-world) duration in garbage collection cycles. |
| `go_gc_gogc_percent` | gauge | `%` | — | restserver | promclient | Heap size target percentage configured by the user, otherwise 100. This value is set by the GOGC environment variable, and the runtime/debug.SetGCPercent function. Sourced from /gc/gogc:percent. |
| `go_gc_gomemlimit_bytes` | gauge | `By` | — | restserver | promclient | Go runtime memory limit configured by the user, otherwise math.MaxInt64. This value is set by the GOMEMLIMIT environment variable, and the runtime/debug.SetMemoryLimit function. Sourced from /gc/gomemlimit:bytes. |
| `go_goroutines` | gauge | `{goroutine}` | — | restserver | promclient | Number of goroutines that currently exist. |
| `go_info` | gauge | — | `version` | restserver | promclient | Information about the Go environment. |
| `go_memstats_alloc_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes allocated in heap and currently in use. Equals to /memory/classes/heap/objects:bytes. |
| `go_memstats_alloc_bytes_total` | counter | `By` | — | restserver | promclient | Total number of bytes allocated in heap until now, even if released already. Equals to /gc/heap/allocs:bytes. |
| `go_memstats_buck_hash_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes used by the profiling bucket hash table. Equals to /memory/classes/profiling/buckets:bytes. |
| `go_memstats_frees_total` | counter | `{object}` | — | restserver | promclient | Total number of heap objects frees. Equals to /gc/heap/frees:objects + /gc/heap/tiny/allocs:objects. |
| `go_memstats_gc_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes used for garbage collection system metadata. Equals to /memory/classes/metadata/other:bytes. |
| `go_memstats_heap_alloc_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes allocated and currently in use, same as go_memstats_alloc_bytes. Equals to /memory/classes/heap/objects:bytes. |
| `go_memstats_heap_idle_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes waiting to be used. Equals to /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes. |
| `go_memstats_heap_inuse_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes that are in use. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes |
| `go_memstats_heap_objects` | gauge | `{object}` | — | restserver | promclient | Number of currently allocated objects. Equals to /gc/heap/objects:objects. |
| `go_memstats_heap_released_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes released to OS. Equals to /memory/classes/heap/released:bytes. |
| `go_memstats_heap_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes obtained from system. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes + /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes. |
| `go_memstats_last_gc_time_seconds` | gauge | `s` | — | restserver | promclient | Number of seconds since 1970 of last garbage collection. |
| `go_memstats_mallocs_total` | counter | `{object}` | — | restserver | promclient | Total number of heap objects allocated, both live and gc-ed. Semantically a counter version for go_memstats_heap_objects gauge. Equals to /gc/heap/allocs:objects + /gc/heap/tiny/allocs:objects. |
| `go_memstats_mcache_inuse_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes in use by mcache structures. Equals to /memory/classes/metadata/mcache/inuse:bytes. |
| `go_memstats_mcache_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes used for mcache structures obtained from system. Equals to /memory/classes/metadata/mcache/inuse:bytes + /memory/classes/metadata/mcache/free:bytes. |
| `go_memstats_mspan_inuse_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes in use by mspan structures. Equals to /memory/classes/metadata/mspan/inuse:bytes. |
| `go_memstats_mspan_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes used for mspan structures obtained from system. Equals to /memory/classes/metadata/mspan/inuse:bytes + /memory/classes/metadata/mspan/free:bytes. |
| `go_memstats_next_gc_bytes` | gauge | `By` | — | restserver | promclient | Number of heap bytes when next garbage collection will take place. Equals to /gc/heap/goal:bytes. |
| `go_memstats_other_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes used for other system allocations. Equals to /memory/classes/other:bytes. |
| `go_memstats_stack_inuse_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes obtained from system for stack allocator in non-CGO environments. Equals to /memory/classes/heap/stacks:bytes. |
| `go_memstats_stack_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes obtained from system for stack allocator. Equals to /memory/classes/heap/stacks:bytes + /memory/classes/os-stacks:bytes. |
| `go_memstats_sys_bytes` | gauge | `By` | — | restserver | promclient | Number of bytes obtained from system. Equals to /memory/classes/total:byte. |
| `go_sched_gomaxprocs_threads` | gauge | `{thread}` | — | restserver | promclient | The current runtime.GOMAXPROCS setting, or the number of operating system threads that can execute user-level Go code simultaneously. Sourced from /sched/gomaxprocs:threads. |
| `go_threads` | gauge | `{thread}` | — | restserver | promclient | Number of OS threads created. |
| `process_cpu_seconds_total` | counter | `s` | — | restserver | promclient | Total user and system CPU time spent in seconds. |
| `process_max_fds` | gauge | `{file_descriptor}` | — | restserver | promclient | Maximum number of open file descriptors. |
| `process_network_receive_bytes_total` | counter | `By` | — | restserver | promclient | Number of bytes received by the process over the network. |
| `process_network_transmit_bytes_total` | counter | `By` | — | restserver | promclient | Number of bytes sent by the process over the network. |
| `process_open_fds` | gauge | `{file_descriptor}` | — | restserver | promclient | Number of open file descriptors. |
| `process_resident_memory_bytes` | gauge | `By` | — | restserver | promclient | Resident memory size in bytes. |
| `process_start_time_seconds` | gauge | `s` | — | restserver | promclient | Start time of the process since unix epoch in seconds. |
| `process_virtual_memory_bytes` | gauge | `By` | — | restserver | promclient | Virtual memory size in bytes. |
| `process_virtual_memory_max_bytes` | gauge | `By` | — | restserver | promclient | Maximum amount of virtual memory available in bytes. |

#### Deprecated, emitted only with `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true`

| Metric | Type | Unit | Attributes | Module | Origin | Description |
|--------|------|------|------------|--------|--------|-------------|
| `runtime.uptime` | observable_counter | `ms` | — | monitoring | runtime | Milliseconds since application was initialized |
| `process.runtime.go.goroutines` | observable_updowncounter | — | — | monitoring | runtime | Number of goroutines that currently exist |
| `process.runtime.go.cgo.calls` | observable_updowncounter | — | — | monitoring | runtime | Number of cgo calls made by the current process |
| `process.runtime.go.mem.heap_alloc` | observable_updowncounter | `By` | — | monitoring | runtime | Bytes of allocated heap objects |
| `process.runtime.go.mem.heap_idle` | observable_updowncounter | `By` | — | monitoring | runtime | Bytes in idle (unused) spans |
| `process.runtime.go.mem.heap_inuse` | observable_updowncounter | `By` | — | monitoring | runtime | Bytes in in-use spans |
| `process.runtime.go.mem.heap_objects` | observable_updowncounter | — | — | monitoring | runtime | Number of allocated heap objects |
| `process.runtime.go.mem.heap_released` | observable_updowncounter | `By` | — | monitoring | runtime | Bytes of idle spans whose physical memory has been returned to the OS |
| `process.runtime.go.mem.heap_sys` | observable_updowncounter | `By` | — | monitoring | runtime | Bytes of heap memory obtained from the OS |
| `process.runtime.go.mem.lookups` | observable_counter | — | — | monitoring | runtime | Number of pointer lookups performed by the runtime |
| `process.runtime.go.mem.live_objects` | observable_updowncounter | — | — | monitoring | runtime | Number of live objects is the number of cumulative Mallocs - Frees |
| `process.runtime.go.gc.count` | observable_counter | — | — | monitoring | runtime | Number of completed garbage collection cycles |
| `process.runtime.go.gc.pause_total_ns` | observable_counter | `ns` | — | monitoring | runtime | Cumulative nanoseconds in GC stop-the-world pauses since the program started |
| `process.runtime.go.gc.pause_ns` | histogram | `ns` | — | monitoring | runtime | Amount of nanoseconds in GC stop-the-world pauses |

<!-- metrics:end -->
