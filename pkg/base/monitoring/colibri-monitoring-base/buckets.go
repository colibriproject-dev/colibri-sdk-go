package colibri_monitoring_base

// DurationBucketsSeconds are the histogram boundaries of every duration recorded in seconds.
// They start with the ones the OpenTelemetry semantic conventions recommend for
// http.server.request.duration, so latency percentiles are meaningful under a second, and go
// on up to an hour, so long waits such as messaging.consume.lag are not all lumped in +Inf.
// The OpenTelemetry defaults (0, 5, 10, 25, ... 10000) are meant for milliseconds and would
// put nearly every request in the first bucket.
var DurationBucketsSeconds = []float64{
	0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
	30, 60, 300, 900, 3600,
}
