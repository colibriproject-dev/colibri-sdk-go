package monitoring

import "slices"

// MetricKind is the instrument type a metric is recorded with. It decides how the metric is
// exposed: counters get the _total suffix on /metrics, up-down counters and gauges do not.
type MetricKind string

const (
	KindCounter                 MetricKind = "counter"
	KindUpDownCounter           MetricKind = "updowncounter"
	KindHistogram               MetricKind = "histogram"
	KindGauge                   MetricKind = "gauge"
	KindObservableCounter       MetricKind = "observable_counter"
	KindObservableUpDownCounter MetricKind = "observable_updowncounter"
	KindObservableGauge         MetricKind = "observable_gauge"
)

// MetricOrigin is who emits a metric: the SDK itself or the third-party instrumentation it
// wires. It makes the origin explicit without encoding it in the name.
type MetricOrigin string

const (
	OriginSDK       MetricOrigin = "sdk"
	OriginOtelHTTP  MetricOrigin = "otelhttp"
	OriginOtelSQL   MetricOrigin = "otelsql"
	OriginRedisOtel MetricOrigin = "redisotel"
	OriginRuntime   MetricOrigin = "runtime"
)

// MetricDefinition describes a metric the SDK emits.
type MetricDefinition struct {
	Name        string
	Description string
	Kind        MetricKind
	// Unit is the UCUM unit the metric is recorded in.
	Unit string
	// Attributes are the keys the metric may carry. SDK metrics carry all of them; a
	// third-party metric may leave some out, e.g. the status code of a failed request.
	Attributes []string
	Origin     MetricOrigin
	// Module is the SDK package that enables the metric.
	Module string
	// Deprecated marks the legacy names kept for backends that still chart them; they are
	// emitted only when opted in.
	Deprecated bool
}

// Names of the metrics emitted by the SDK itself.
const (
	MetricHTTPServerRequestDuration  = "http.server.request.duration"
	MetricHTTPServerActiveRequests   = "http.server.active_requests"
	MetricHTTPServerRequestBodySize  = "http.server.request.body.size"
	MetricHTTPServerResponseBodySize = "http.server.response.body.size"
	MetricHTTPServerPanicRecovered   = "http.server.panic.recovered"

	MetricMessagingPublished       = "messaging.published"
	MetricMessagingConsumed        = "messaging.consumed"
	MetricMessagingProcessDuration = "messaging.process.duration"
	MetricMessagingRejected        = "messaging.rejected"
	MetricMessagingInFlight        = "messaging.in_flight"

	MetricStorageOperation         = "storage.operation"
	MetricStorageOperationDuration = "storage.operation.duration"
	MetricStorageTransferred       = "storage.transferred"
)

// Names of the metrics emitted by the third-party instrumentation the SDK wires.
const (
	MetricHTTPClientRequestDuration = "http.client.request.duration"
	MetricHTTPClientRequestBodySize = "http.client.request.body.size"

	MetricDBSQLClientLatency             = "db.sql.client.latency"
	MetricDBSQLClientCalls               = "db.sql.client.calls"
	MetricDBSQLConnectionsOpen           = "db.sql.connections.open"
	MetricDBSQLConnectionsIdle           = "db.sql.connections.idle"
	MetricDBSQLConnectionsActive         = "db.sql.connections.active"
	MetricDBSQLConnectionsWaitCount      = "db.sql.connections.wait_count"
	MetricDBSQLConnectionsWaitDuration   = "db.sql.connections.wait_duration"
	MetricDBSQLConnectionsIdleClosed     = "db.sql.connections.idle_closed"
	MetricDBSQLConnectionsIdleTimeClosed = "db.sql.connections.idle_time_closed"
	MetricDBSQLConnectionsLifetimeClosed = "db.sql.connections.lifetime_closed"

	MetricDBClientConnectionsIdleMax       = "db.client.connections.idle.max"
	MetricDBClientConnectionsIdleMin       = "db.client.connections.idle.min"
	MetricDBClientConnectionsMax           = "db.client.connections.max"
	MetricDBClientConnectionsUsage         = "db.client.connections.usage"
	MetricDBClientConnectionsWaits         = "db.client.connections.waits"
	MetricDBClientConnectionsWaitsDuration = "db.client.connections.waits_duration"
	MetricDBClientConnectionsTimeouts      = "db.client.connections.timeouts"
	MetricDBClientConnectionsHits          = "db.client.connections.hits"
	MetricDBClientConnectionsMisses        = "db.client.connections.misses"
	MetricDBClientConnectionsCreateTime    = "db.client.connections.create_time"
	MetricDBClientConnectionsUseTime       = "db.client.connections.use_time"

	MetricGoMemoryUsed        = "go.memory.used"
	MetricGoMemoryLimit       = "go.memory.limit"
	MetricGoMemoryAllocated   = "go.memory.allocated"
	MetricGoMemoryAllocations = "go.memory.allocations"
	MetricGoMemoryGCGoal      = "go.memory.gc.goal"
	MetricGoGoroutineCount    = "go.goroutine.count"
	MetricGoProcessorLimit    = "go.processor.limit"
	MetricGoConfigGogc        = "go.config.gogc"
)

var (
	httpServerRequestAttrs  = []string{"http.request.method", "server.address", "url.scheme"}
	httpServerResponseAttrs = []string{"http.request.method", "http.response.status_code", "http.route", "server.address", "url.scheme"}
	httpClientAttrs         = []string{
		"http.request.method", "http.response.status_code", "network.protocol.name",
		"network.protocol.version", "server.address", "server.port", "url.scheme",
	}

	// db.sql.error carries the error message, see the cardinality notes in
	// docs/observability/metrics.md.
	sqlClientAttrs = []string{"db.name", "db.operation", "db.sql.error", "db.sql.status", "db.system.name"}
	sqlPoolAttrs   = []string{"db.instance", "db.system.name"}

	redisPoolAttrs = []string{"db.system", "pool.name"}
)

// catalog is every metric the SDK emits, with or without opting in. It is kept in the order
// the reference doc lists it.
var catalog = []MetricDefinition{
	// ── restserver ────────────────────────────────────────────────────────────
	sdkMetric(MetricHTTPServerRequestDuration, "Duration of HTTP server requests",
		KindHistogram, "s", "restserver", httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerActiveRequests, "Number of active HTTP server requests",
		KindUpDownCounter, "{request}", "restserver", httpServerRequestAttrs...),
	sdkMetric(MetricHTTPServerRequestBodySize, "Size of HTTP server request bodies",
		KindHistogram, "By", "restserver", httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerResponseBodySize, "Size of HTTP server response bodies",
		KindHistogram, "By", "restserver", httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerPanicRecovered, "Number of panics recovered while serving HTTP requests",
		KindCounter, "{panic}", "restserver", "http.request.method", "http.route"),

	// ── messaging ─────────────────────────────────────────────────────────────
	sdkMetric(MetricMessagingPublished, "Number of messages published",
		KindCounter, "{message}", "messaging", "topic", "result"),
	sdkMetric(MetricMessagingConsumed, "Number of messages consumed",
		KindCounter, "{message}", "messaging", "queue", "action", "result"),
	sdkMetric(MetricMessagingProcessDuration, "Duration of the processing of a consumed message",
		KindHistogram, "s", "messaging", "queue", "action", "result"),
	sdkMetric(MetricMessagingRejected,
		"Number of consumed messages rejected without requeue, left to the broker dead-letter handling",
		KindCounter, "{message}", "messaging", "queue", "action", "reason"),
	sdkMetric(MetricMessagingInFlight, "Number of messages being processed",
		KindObservableGauge, "{message}", "messaging", "queue"),

	// ── storage ───────────────────────────────────────────────────────────────
	sdkMetric(MetricStorageOperation, "Number of storage operations",
		KindCounter, "{operation}", "storage", "operation", "result"),
	sdkMetric(MetricStorageOperationDuration, "Duration of storage operations",
		KindHistogram, "s", "storage", "operation", "result"),
	sdkMetric(MetricStorageTransferred, "Size of the files uploaded to and downloaded from the storage",
		KindHistogram, "By", "storage", "operation"),

	// ── restclient, via otelhttp ──────────────────────────────────────────────
	{Name: MetricHTTPClientRequestDuration, Description: "Duration of HTTP client requests.",
		Kind: KindHistogram, Unit: "s", Attributes: httpClientAttrs, Origin: OriginOtelHTTP, Module: "restclient"},
	{Name: MetricHTTPClientRequestBodySize, Description: "Size of HTTP client request bodies.",
		Kind: KindHistogram, Unit: "By", Attributes: httpClientAttrs, Origin: OriginOtelHTTP, Module: "restclient"},

	// ── sqlDB, via otelsql ────────────────────────────────────────────────────
	sqlMetric(MetricDBSQLClientLatency, "The distribution of latencies of various calls in milliseconds",
		KindHistogram, "ms", sqlClientAttrs),
	sqlMetric(MetricDBSQLClientCalls, "The number of various calls of methods",
		KindCounter, "1", sqlClientAttrs),
	sqlMetric(MetricDBSQLConnectionsOpen, "Count of open connections in the pool",
		KindObservableGauge, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdle, "Count of idle connections in the pool",
		KindObservableGauge, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsActive, "Count of active connections in the pool",
		KindObservableGauge, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsWaitCount, "The total number of connections waited for",
		KindObservableCounter, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsWaitDuration, "The total time blocked waiting for a new connection",
		KindObservableCounter, "ms", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdleClosed, "The total number of connections closed due to SetMaxIdleConns",
		KindObservableCounter, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdleTimeClosed, "The total number of connections closed due to SetConnMaxIdleTime",
		KindObservableCounter, "1", sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsLifetimeClosed, "The total number of connections closed due to SetConnMaxLifetime",
		KindObservableCounter, "1", sqlPoolAttrs),

	// ── cacheDB, via redisotel ────────────────────────────────────────────────
	redisMetric(MetricDBClientConnectionsIdleMax, "The maximum number of idle open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsIdleMin, "The minimum number of idle open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsMax, "The maximum number of open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsUsage,
		"The number of connections that are currently in state described by the state attribute",
		KindObservableUpDownCounter, "", "db.system", "pool.name", "state"),
	redisMetric(MetricDBClientConnectionsWaits, "The number of times a connection was waited for",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsWaitsDuration, "The total time spent for waiting a connection in nanoseconds",
		KindObservableUpDownCounter, "ns", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsTimeouts,
		"The number of connection timeouts that have occurred trying to obtain a connection from the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsHits, "The number of times free connection was found in the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsMisses, "The number of times free connection was not found in the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsCreateTime, "The time it took to create a new connection.",
		KindHistogram, "ms", "db.system", "error_type", "pool.name", "status"),
	redisMetric(MetricDBClientConnectionsUseTime,
		"The time between borrowing a connection and returning it to the pool.",
		KindHistogram, "ms", "db.system", "error_type", "pool.name", "status", "type"),

	// ── Go runtime ────────────────────────────────────────────────────────────
	runtimeMetric(MetricGoMemoryUsed, "Memory used by the Go runtime.",
		KindObservableUpDownCounter, "By", "go.memory.type"),
	runtimeMetric(MetricGoMemoryLimit, "Go runtime memory limit configured by the user, if a limit exists.",
		KindObservableUpDownCounter, "By"),
	runtimeMetric(MetricGoMemoryAllocated, "Memory allocated to the heap by the application.",
		KindObservableCounter, "By"),
	runtimeMetric(MetricGoMemoryAllocations, "Count of allocations to the heap by the application.",
		KindObservableCounter, "{allocation}"),
	runtimeMetric(MetricGoMemoryGCGoal, "Heap size target for the end of the GC cycle.",
		KindObservableUpDownCounter, "By"),
	runtimeMetric(MetricGoGoroutineCount, "Count of live goroutines.",
		KindObservableUpDownCounter, "{goroutine}"),
	runtimeMetric(MetricGoProcessorLimit,
		"The number of OS threads that can execute user-level Go code simultaneously.",
		KindObservableUpDownCounter, "{thread}"),
	runtimeMetric(MetricGoConfigGogc, "Heap size target percentage configured by the user, otherwise 100.",
		KindObservableUpDownCounter, "%"),

	// Legacy runtime names, emitted only with OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true.
	deprecatedRuntimeMetric("runtime.uptime", "Milliseconds since application was initialized",
		KindObservableCounter, "ms"),
	deprecatedRuntimeMetric("process.runtime.go.goroutines", "Number of goroutines that currently exist",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.cgo.calls", "Number of cgo calls made by the current process",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_alloc", "Bytes of allocated heap objects",
		KindObservableUpDownCounter, "By"),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_idle", "Bytes in idle (unused) spans",
		KindObservableUpDownCounter, "By"),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_inuse", "Bytes in in-use spans",
		KindObservableUpDownCounter, "By"),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_objects", "Number of allocated heap objects",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_released",
		"Bytes of idle spans whose physical memory has been returned to the OS",
		KindObservableUpDownCounter, "By"),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_sys", "Bytes of heap memory obtained from the OS",
		KindObservableUpDownCounter, "By"),
	deprecatedRuntimeMetric("process.runtime.go.mem.lookups", "Number of pointer lookups performed by the runtime",
		KindObservableCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.live_objects",
		"Number of live objects is the number of cumulative Mallocs - Frees",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.gc.count", "Number of completed garbage collection cycles",
		KindObservableCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.gc.pause_total_ns",
		"Cumulative nanoseconds in GC stop-the-world pauses since the program started",
		KindObservableCounter, "ns"),
	deprecatedRuntimeMetric("process.runtime.go.gc.pause_ns", "Amount of nanoseconds in GC stop-the-world pauses",
		KindHistogram, "ns"),
}

func sdkMetric(name, description string, kind MetricKind, unit, module string, attributes ...string) MetricDefinition {
	return MetricDefinition{Name: name, Description: description, Kind: kind, Unit: unit,
		Attributes: attributes, Origin: OriginSDK, Module: module}
}

func sqlMetric(name, description string, kind MetricKind, unit string, attributes []string) MetricDefinition {
	return MetricDefinition{Name: name, Description: description, Kind: kind, Unit: unit,
		Attributes: attributes, Origin: OriginOtelSQL, Module: "sqlDB"}
}

func redisMetric(name, description string, kind MetricKind, unit string, attributes ...string) MetricDefinition {
	return MetricDefinition{Name: name, Description: description, Kind: kind, Unit: unit,
		Attributes: attributes, Origin: OriginRedisOtel, Module: "cacheDB"}
}

func runtimeMetric(name, description string, kind MetricKind, unit string, attributes ...string) MetricDefinition {
	return MetricDefinition{Name: name, Description: description, Kind: kind, Unit: unit,
		Attributes: attributes, Origin: OriginRuntime, Module: "monitoring"}
}

func deprecatedRuntimeMetric(name, description string, kind MetricKind, unit string) MetricDefinition {
	m := runtimeMetric(name, description, kind, unit)
	m.Deprecated = true
	return m
}

// Catalog returns every metric the SDK emits. The result is a copy: changing it does not
// change the catalog.
func Catalog() []MetricDefinition {
	definitions := make([]MetricDefinition, len(catalog))
	for i, m := range catalog {
		m.Attributes = slices.Clone(m.Attributes)
		definitions[i] = m
	}

	return definitions
}

// LookupMetric returns the definition of the named metric, and whether the SDK emits it.
func LookupMetric(name string) (MetricDefinition, bool) {
	i := slices.IndexFunc(catalog, func(m MetricDefinition) bool { return m.Name == name })
	if i < 0 {
		return MetricDefinition{}, false
	}

	m := catalog[i]
	m.Attributes = slices.Clone(m.Attributes)
	return m, true
}
