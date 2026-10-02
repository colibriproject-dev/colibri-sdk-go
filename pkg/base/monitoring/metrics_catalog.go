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
	// KindSummary is a Prometheus summary, only exposed by the Prometheus client collectors.
	KindSummary MetricKind = "summary"
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
	// OriginPromClient marks the Go and process collectors of the Prometheus client, which
	// /metrics serves next to the OpenTelemetry metrics. They never reach the OTLP exporter,
	// and their name is the Prometheus name, not translated from an OpenTelemetry one.
	OriginPromClient MetricOrigin = "promclient"
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
	MetricMessagingConsumeLag      = "messaging.consume.lag"

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

// Names of the Go and process collectors of the Prometheus client used by the dashboards and
// alert rules. The rest of them are cataloged by name only.
const (
	MetricPromGoGCDurationSeconds         = "go_gc_duration_seconds"
	MetricPromProcessResidentMemoryBytes  = "process_resident_memory_bytes"
	MetricPromProcessCPUSecondsTotal      = "process_cpu_seconds_total"
	MetricPromProcessOpenFDs              = "process_open_fds"
	MetricPromProcessMaxFDs               = "process_max_fds"
	MetricPromGoMemstatsHeapInuseBytes    = "go_memstats_heap_inuse_bytes"
	MetricPromGoMemstatsNextGCBytes       = "go_memstats_next_gc_bytes"
	MetricPromGoMemstatsAllocBytesTotal   = "go_memstats_alloc_bytes_total"
	MetricPromGoMemstatsLastGCTimeSeconds = "go_memstats_last_gc_time_seconds"
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
	sdkMetric(MetricMessagingConsumeLag, "Time between the broker accepting a message and a consumer receiving it",
		KindHistogram, unitSeconds, moduleMessaging, attrQueue, attrAction),

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

	// ── Prometheus client collectors, only on /metrics ────────────────────────
	promClientMetric(MetricPromGoGCDurationSeconds,
		"A summary of the wall-time pause (stop-the-world) duration in garbage collection cycles.",
		KindSummary, unitSeconds, "quantile"),
	promClientMetric("go_gc_gogc_percent",
		"Heap size target percentage configured by the user, otherwise 100. This value is set by the GOGC environment variable, and the runtime/debug.SetGCPercent function. Sourced from /gc/gogc:percent.",
		KindGauge, "%"),
	promClientMetric("go_gc_gomemlimit_bytes",
		"Go runtime memory limit configured by the user, otherwise math.MaxInt64. This value is set by the GOMEMLIMIT environment variable, and the runtime/debug.SetMemoryLimit function. Sourced from /gc/gomemlimit:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_goroutines", "Number of goroutines that currently exist.",
		KindGauge, "{goroutine}"),
	promClientMetric("go_info", "Information about the Go environment.",
		KindGauge, "", "version"),
	promClientMetric("go_memstats_alloc_bytes",
		"Number of bytes allocated in heap and currently in use. Equals to /memory/classes/heap/objects:bytes.",
		KindGauge, unitBytes),
	promClientMetric(MetricPromGoMemstatsAllocBytesTotal,
		"Total number of bytes allocated in heap until now, even if released already. Equals to /gc/heap/allocs:bytes.",
		KindCounter, unitBytes),
	promClientMetric("go_memstats_buck_hash_sys_bytes",
		"Number of bytes used by the profiling bucket hash table. Equals to /memory/classes/profiling/buckets:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_frees_total",
		"Total number of heap objects frees. Equals to /gc/heap/frees:objects + /gc/heap/tiny/allocs:objects.",
		KindCounter, "{object}"),
	promClientMetric("go_memstats_gc_sys_bytes",
		"Number of bytes used for garbage collection system metadata. Equals to /memory/classes/metadata/other:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_heap_alloc_bytes",
		"Number of heap bytes allocated and currently in use, same as go_memstats_alloc_bytes. Equals to /memory/classes/heap/objects:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_heap_idle_bytes",
		"Number of heap bytes waiting to be used. Equals to /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.",
		KindGauge, unitBytes),
	promClientMetric(MetricPromGoMemstatsHeapInuseBytes,
		"Number of heap bytes that are in use. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_heap_objects",
		"Number of currently allocated objects. Equals to /gc/heap/objects:objects.",
		KindGauge, "{object}"),
	promClientMetric("go_memstats_heap_released_bytes",
		"Number of heap bytes released to OS. Equals to /memory/classes/heap/released:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_heap_sys_bytes",
		"Number of heap bytes obtained from system. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes + /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.",
		KindGauge, unitBytes),
	promClientMetric(MetricPromGoMemstatsLastGCTimeSeconds,
		"Number of seconds since 1970 of last garbage collection.",
		KindGauge, unitSeconds),
	promClientMetric("go_memstats_mallocs_total",
		"Total number of heap objects allocated, both live and gc-ed. Semantically a counter version for go_memstats_heap_objects gauge. Equals to /gc/heap/allocs:objects + /gc/heap/tiny/allocs:objects.",
		KindCounter, "{object}"),
	promClientMetric("go_memstats_mcache_inuse_bytes",
		"Number of bytes in use by mcache structures. Equals to /memory/classes/metadata/mcache/inuse:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_mcache_sys_bytes",
		"Number of bytes used for mcache structures obtained from system. Equals to /memory/classes/metadata/mcache/inuse:bytes + /memory/classes/metadata/mcache/free:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_mspan_inuse_bytes",
		"Number of bytes in use by mspan structures. Equals to /memory/classes/metadata/mspan/inuse:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_mspan_sys_bytes",
		"Number of bytes used for mspan structures obtained from system. Equals to /memory/classes/metadata/mspan/inuse:bytes + /memory/classes/metadata/mspan/free:bytes.",
		KindGauge, unitBytes),
	promClientMetric(MetricPromGoMemstatsNextGCBytes,
		"Number of heap bytes when next garbage collection will take place. Equals to /gc/heap/goal:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_other_sys_bytes",
		"Number of bytes used for other system allocations. Equals to /memory/classes/other:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_stack_inuse_bytes",
		"Number of bytes obtained from system for stack allocator in non-CGO environments. Equals to /memory/classes/heap/stacks:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_stack_sys_bytes",
		"Number of bytes obtained from system for stack allocator. Equals to /memory/classes/heap/stacks:bytes + /memory/classes/os-stacks:bytes.",
		KindGauge, unitBytes),
	promClientMetric("go_memstats_sys_bytes",
		"Number of bytes obtained from system. Equals to /memory/classes/total:byte.",
		KindGauge, unitBytes),
	promClientMetric("go_sched_gomaxprocs_threads",
		"The current runtime.GOMAXPROCS setting, or the number of operating system threads that can execute user-level Go code simultaneously. Sourced from /sched/gomaxprocs:threads.",
		KindGauge, "{thread}"),
	promClientMetric("go_threads", "Number of OS threads created.",
		KindGauge, "{thread}"),
	promClientMetric(MetricPromProcessCPUSecondsTotal, "Total user and system CPU time spent in seconds.",
		KindCounter, unitSeconds),
	promClientMetric(MetricPromProcessMaxFDs, "Maximum number of open file descriptors.",
		KindGauge, "{file_descriptor}"),
	promClientMetric("process_network_receive_bytes_total",
		"Number of bytes received by the process over the network.",
		KindCounter, unitBytes),
	promClientMetric("process_network_transmit_bytes_total",
		"Number of bytes sent by the process over the network.",
		KindCounter, unitBytes),
	promClientMetric(MetricPromProcessOpenFDs, "Number of open file descriptors.",
		KindGauge, "{file_descriptor}"),
	promClientMetric(MetricPromProcessResidentMemoryBytes, "Resident memory size in bytes.",
		KindGauge, unitBytes),
	promClientMetric("process_start_time_seconds", "Start time of the process since unix epoch in seconds.",
		KindGauge, unitSeconds),
	promClientMetric("process_virtual_memory_bytes", "Virtual memory size in bytes.",
		KindGauge, unitBytes),
	promClientMetric("process_virtual_memory_max_bytes", "Maximum amount of virtual memory available in bytes.",
		KindGauge, unitBytes),
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

func promClientMetric(name, description string, kind MetricKind, unit string, attributes ...string) MetricDefinition {
	return MetricDefinition{Name: name, Description: description, Kind: kind, Unit: unit,
		Attributes: attributes, Origin: OriginPromClient, Module: moduleRestServer}
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
