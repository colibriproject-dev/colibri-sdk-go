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

// Attribute keys, units and modules shared by several catalog entries.
const (
	attrHTTPRequestMethod      = "http.request.method"
	attrHTTPResponseStatusCode = "http.response.status_code"
	attrHTTPRoute              = "http.route"
	attrServerAddress          = "server.address"
	attrURLScheme              = "url.scheme"
	attrDBSystem               = "db.system"
	attrDBSystemName           = "db.system.name"
	attrPoolName               = "pool.name"
	attrErrorType              = "error_type"
	attrStatus                 = "status"
	attrQueue                  = "queue"
	attrAction                 = "action"
	attrResult                 = "result"
	attrOperation              = "operation"

	unitSeconds       = "s"
	unitMilliseconds  = "ms"
	unitNanoseconds   = "ns"
	unitBytes         = "By"
	unitDimensionless = "1"
	unitMessage       = "{message}"

	moduleRestServer = "restserver"
	moduleRestClient = "restclient"
	moduleMessaging  = "messaging"
	moduleStorage    = "storage"
)

var (
	httpServerRequestAttrs  = []string{attrHTTPRequestMethod, attrServerAddress, attrURLScheme}
	httpServerResponseAttrs = []string{attrHTTPRequestMethod, attrHTTPResponseStatusCode, attrHTTPRoute, attrServerAddress, attrURLScheme}
	httpClientAttrs         = []string{
		attrHTTPRequestMethod, attrHTTPResponseStatusCode, "network.protocol.name",
		"network.protocol.version", attrServerAddress, "server.port", attrURLScheme,
	}

	// db.sql.error carries the error message, see the cardinality notes in
	// docs/observability/metrics.md.
	sqlClientAttrs = []string{"db.name", "db.operation", "db.sql.error", "db.sql.status", attrDBSystemName}
	sqlPoolAttrs   = []string{"db.instance", attrDBSystemName}

	redisPoolAttrs = []string{attrDBSystem, attrPoolName}
)

// catalog is every metric the SDK emits, with or without opting in. It is kept in the order
// the reference doc lists it.
var catalog = []MetricDefinition{
	// ── restserver ────────────────────────────────────────────────────────────
	sdkMetric(MetricHTTPServerRequestDuration, "Duration of HTTP server requests",
		KindHistogram, unitSeconds, moduleRestServer, httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerActiveRequests, "Number of active HTTP server requests",
		KindUpDownCounter, "{request}", moduleRestServer, httpServerRequestAttrs...),
	sdkMetric(MetricHTTPServerRequestBodySize, "Size of HTTP server request bodies",
		KindHistogram, unitBytes, moduleRestServer, httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerResponseBodySize, "Size of HTTP server response bodies",
		KindHistogram, unitBytes, moduleRestServer, httpServerResponseAttrs...),
	sdkMetric(MetricHTTPServerPanicRecovered, "Number of panics recovered while serving HTTP requests",
		KindCounter, "{panic}", moduleRestServer, attrHTTPRequestMethod, attrHTTPRoute),

	// ── messaging ─────────────────────────────────────────────────────────────
	sdkMetric(MetricMessagingPublished, "Number of messages published",
		KindCounter, unitMessage, moduleMessaging, "topic", attrResult),
	sdkMetric(MetricMessagingConsumed, "Number of messages consumed",
		KindCounter, unitMessage, moduleMessaging, attrQueue, attrAction, attrResult),
	sdkMetric(MetricMessagingProcessDuration, "Duration of the processing of a consumed message",
		KindHistogram, unitSeconds, moduleMessaging, attrQueue, attrAction, attrResult),
	sdkMetric(MetricMessagingRejected,
		"Number of consumed messages rejected without requeue, left to the broker dead-letter handling",
		KindCounter, unitMessage, moduleMessaging, attrQueue, attrAction, "reason"),
	sdkMetric(MetricMessagingInFlight, "Number of messages being processed",
		KindObservableGauge, unitMessage, moduleMessaging, attrQueue),

	// ── storage ───────────────────────────────────────────────────────────────
	sdkMetric(MetricStorageOperation, "Number of storage operations",
		KindCounter, "{operation}", moduleStorage, attrOperation, attrResult),
	sdkMetric(MetricStorageOperationDuration, "Duration of storage operations",
		KindHistogram, unitSeconds, moduleStorage, attrOperation, attrResult),
	sdkMetric(MetricStorageTransferred, "Size of the files uploaded to and downloaded from the storage",
		KindHistogram, unitBytes, moduleStorage, attrOperation),

	// ── restclient, via otelhttp ──────────────────────────────────────────────
	{Name: MetricHTTPClientRequestDuration, Description: "Duration of HTTP client requests.",
		Kind: KindHistogram, Unit: unitSeconds, Attributes: httpClientAttrs, Origin: OriginOtelHTTP, Module: moduleRestClient},
	{Name: MetricHTTPClientRequestBodySize, Description: "Size of HTTP client request bodies.",
		Kind: KindHistogram, Unit: unitBytes, Attributes: httpClientAttrs, Origin: OriginOtelHTTP, Module: moduleRestClient},

	// ── sqlDB, via otelsql ────────────────────────────────────────────────────
	sqlMetric(MetricDBSQLClientLatency, "The distribution of latencies of various calls in milliseconds",
		KindHistogram, unitMilliseconds, sqlClientAttrs),
	sqlMetric(MetricDBSQLClientCalls, "The number of various calls of methods",
		KindCounter, unitDimensionless, sqlClientAttrs),
	sqlMetric(MetricDBSQLConnectionsOpen, "Count of open connections in the pool",
		KindObservableGauge, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdle, "Count of idle connections in the pool",
		KindObservableGauge, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsActive, "Count of active connections in the pool",
		KindObservableGauge, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsWaitCount, "The total number of connections waited for",
		KindObservableCounter, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsWaitDuration, "The total time blocked waiting for a new connection",
		KindObservableCounter, unitMilliseconds, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdleClosed, "The total number of connections closed due to SetMaxIdleConns",
		KindObservableCounter, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsIdleTimeClosed, "The total number of connections closed due to SetConnMaxIdleTime",
		KindObservableCounter, unitDimensionless, sqlPoolAttrs),
	sqlMetric(MetricDBSQLConnectionsLifetimeClosed, "The total number of connections closed due to SetConnMaxLifetime",
		KindObservableCounter, unitDimensionless, sqlPoolAttrs),

	// ── cacheDB, via redisotel ────────────────────────────────────────────────
	redisMetric(MetricDBClientConnectionsIdleMax, "The maximum number of idle open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsIdleMin, "The minimum number of idle open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsMax, "The maximum number of open connections allowed",
		KindObservableUpDownCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsUsage,
		"The number of connections that are currently in state described by the state attribute",
		KindObservableUpDownCounter, "", attrDBSystem, attrPoolName, "state"),
	redisMetric(MetricDBClientConnectionsWaits, "The number of times a connection was waited for",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsWaitsDuration, "The total time spent for waiting a connection in nanoseconds",
		KindObservableUpDownCounter, unitNanoseconds, redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsTimeouts,
		"The number of connection timeouts that have occurred trying to obtain a connection from the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsHits, "The number of times free connection was found in the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsMisses, "The number of times free connection was not found in the pool",
		KindObservableCounter, "", redisPoolAttrs...),
	redisMetric(MetricDBClientConnectionsCreateTime, "The time it took to create a new connection.",
		KindHistogram, unitMilliseconds, attrDBSystem, attrErrorType, attrPoolName, attrStatus),
	redisMetric(MetricDBClientConnectionsUseTime,
		"The time between borrowing a connection and returning it to the pool.",
		KindHistogram, unitMilliseconds, attrDBSystem, attrErrorType, attrPoolName, attrStatus, "type"),

	// ── Go runtime ────────────────────────────────────────────────────────────
	runtimeMetric(MetricGoMemoryUsed, "Memory used by the Go runtime.",
		KindObservableUpDownCounter, unitBytes, "go.memory.type"),
	runtimeMetric(MetricGoMemoryLimit, "Go runtime memory limit configured by the user, if a limit exists.",
		KindObservableUpDownCounter, unitBytes),
	runtimeMetric(MetricGoMemoryAllocated, "Memory allocated to the heap by the application.",
		KindObservableCounter, unitBytes),
	runtimeMetric(MetricGoMemoryAllocations, "Count of allocations to the heap by the application.",
		KindObservableCounter, "{allocation}"),
	runtimeMetric(MetricGoMemoryGCGoal, "Heap size target for the end of the GC cycle.",
		KindObservableUpDownCounter, unitBytes),
	runtimeMetric(MetricGoGoroutineCount, "Count of live goroutines.",
		KindObservableUpDownCounter, "{goroutine}"),
	runtimeMetric(MetricGoProcessorLimit,
		"The number of OS threads that can execute user-level Go code simultaneously.",
		KindObservableUpDownCounter, "{thread}"),
	runtimeMetric(MetricGoConfigGogc, "Heap size target percentage configured by the user, otherwise 100.",
		KindObservableUpDownCounter, "%"),

	// Legacy runtime names, emitted only with OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true.
	deprecatedRuntimeMetric("runtime.uptime", "Milliseconds since application was initialized",
		KindObservableCounter, unitMilliseconds),
	deprecatedRuntimeMetric("process.runtime.go.goroutines", "Number of goroutines that currently exist",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.cgo.calls", "Number of cgo calls made by the current process",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_alloc", "Bytes of allocated heap objects",
		KindObservableUpDownCounter, unitBytes),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_idle", "Bytes in idle (unused) spans",
		KindObservableUpDownCounter, unitBytes),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_inuse", "Bytes in in-use spans",
		KindObservableUpDownCounter, unitBytes),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_objects", "Number of allocated heap objects",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_released",
		"Bytes of idle spans whose physical memory has been returned to the OS",
		KindObservableUpDownCounter, unitBytes),
	deprecatedRuntimeMetric("process.runtime.go.mem.heap_sys", "Bytes of heap memory obtained from the OS",
		KindObservableUpDownCounter, unitBytes),
	deprecatedRuntimeMetric("process.runtime.go.mem.lookups", "Number of pointer lookups performed by the runtime",
		KindObservableCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.mem.live_objects",
		"Number of live objects is the number of cumulative Mallocs - Frees",
		KindObservableUpDownCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.gc.count", "Number of completed garbage collection cycles",
		KindObservableCounter, ""),
	deprecatedRuntimeMetric("process.runtime.go.gc.pause_total_ns",
		"Cumulative nanoseconds in GC stop-the-world pauses since the program started",
		KindObservableCounter, unitNanoseconds),
	deprecatedRuntimeMetric("process.runtime.go.gc.pause_ns", "Amount of nanoseconds in GC stop-the-world pauses",
		KindHistogram, unitNanoseconds),
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
