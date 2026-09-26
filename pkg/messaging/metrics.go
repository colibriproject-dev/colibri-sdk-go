package messaging

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
)

// Metric names and attributes emitted by the messaging module. They move to the SDK metric
// catalog once it exists (#233).
//
// correlationId and messageId are deliberately absent: they are unbounded and belong on the
// spans only. action is set by the application and must come from a fixed set.
const (
	metricPublished       = "messaging.published"
	metricConsumed        = "messaging.consumed"
	metricProcessDuration = "messaging.process.duration"
	metricRejected        = "messaging.rejected"
	metricInFlight        = "messaging.in_flight"

	attrTopic  = "topic"
	attrQueue  = "queue"
	attrAction = "action"
	attrResult = "result"
	attrReason = "reason"

	resultSuccess = "success"
	resultError   = "error"
	resultPanic   = "panic"
)

// messagingMetrics holds the instruments of the module, created once per Initialize.
type messagingMetrics struct {
	published       colibrimonitoringbase.Counter
	consumed        colibrimonitoringbase.Counter
	processDuration colibrimonitoringbase.HistogramRecorder
	rejected        colibrimonitoringbase.Counter
	inFlight        colibrimonitoringbase.Registration
}

// moduleMetrics is read from the consumer goroutines and the producers while Initialize may
// be replacing it, hence the atomic pointer.
var moduleMetrics atomic.Pointer[messagingMetrics]

// noopMetrics is what the module records into before Initialize has created the instruments.
var noopMetrics = &messagingMetrics{
	published:       colibrimonitoringbase.NoopCounter(),
	consumed:        colibrimonitoringbase.NoopCounter(),
	processDuration: colibrimonitoringbase.NoopHistogram(),
	rejected:        colibrimonitoringbase.NoopCounter(),
	inFlight:        colibrimonitoringbase.NoopRegistration(),
}

// initMetrics creates the instruments of the module, releasing the in-flight callback of a
// previous Initialize so it is not observed twice.
func initMetrics() {
	m := &messagingMetrics{
		published: monitoring.Counter(metricPublished,
			"Number of messages published", "{message}"),
		consumed: monitoring.Counter(metricConsumed,
			"Number of messages consumed", "{message}"),
		processDuration: monitoring.Histogram(metricProcessDuration,
			"Duration of the processing of a consumed message", "s"),
		rejected: monitoring.Counter(metricRejected,
			"Number of consumed messages rejected without requeue, left to the broker dead-letter handling", "{message}"),
		inFlight: monitoring.ObservableGauge(metricInFlight,
			"Number of messages being processed", "{message}", observeInFlight),
	}

	if previous := moduleMetrics.Swap(m); previous != nil {
		unregisterInFlight(previous)
	}
}

// releaseMetrics stops the in-flight callback once the module is closed.
func releaseMetrics() {
	if previous := moduleMetrics.Swap(nil); previous != nil {
		unregisterInFlight(previous)
	}
}

func unregisterInFlight(m *messagingMetrics) {
	if err := m.inFlight.Unregister(); err != nil {
		logging.Warn(context.Background()).Msgf("Unregistering %s: %v", metricInFlight, err)
	}
}

func currentMetrics() *messagingMetrics {
	if m := moduleMetrics.Load(); m != nil {
		return m
	}

	return noopMetrics
}

// observeInFlight reports the messages being processed per queue. Every queue with a
// running consumer is reported, idle ones as zero, so the series does not vanish between
// messages.
func observeInFlight(context.Context) []colibrimonitoringbase.Observation {
	moduleMu.RLock()
	defer moduleMu.RUnlock()

	observations := make([]colibrimonitoringbase.Observation, 0, len(consumers))
	index := make(map[string]int, len(consumers))
	for _, c := range consumers {
		value := float64(c.inFlight.Load())
		if i, ok := index[c.queue]; ok {
			observations[i].Value += value
			continue
		}

		index[c.queue] = len(observations)
		observations = append(observations, colibrimonitoringbase.Observation{
			Value:      value,
			Attributes: c.metricAttrs.queue,
		})
	}

	return observations
}

// recordPublished counts a publish attempt.
func (p *Producer) recordPublished(ctx context.Context, err error) {
	attrs := p.metricAttrs.success
	if err != nil {
		attrs = p.metricAttrs.failure
	}

	currentMetrics().published.AddAttrs(ctx, 1, attrs)
}

// recordConsumed counts a processed message and its processing time.
func (c *consumer) recordConsumed(ctx context.Context, action, result string, elapsed time.Duration) {
	attrs := c.metricAttrs.consumed(action, result)
	m := currentMetrics()
	m.consumed.AddAttrs(ctx, 1, attrs)
	m.processDuration.RecordAttrs(ctx, elapsed.Seconds(), attrs)
}

// recordRejected counts a message rejected without requeue.
func (c *consumer) recordRejected(ctx context.Context, action, reason string) {
	currentMetrics().rejected.AddAttrs(ctx, 1, c.metricAttrs.rejected(action, reason))
}

// producerAttrs are the attribute sets of a producer, built once since its topic is fixed.
type producerAttrs struct {
	success colibrimonitoringbase.Attrs
	failure colibrimonitoringbase.Attrs
}

func newProducerAttrs(topic string) producerAttrs {
	return producerAttrs{
		success: colibrimonitoringbase.NewAttrs(attrTopic, topic, attrResult, resultSuccess),
		failure: colibrimonitoringbase.NewAttrs(attrTopic, topic, attrResult, resultError),
	}
}

// consumerAttrs caches the attribute sets of a consumer. The queue is fixed and the action
// comes from a bounded set, so the cache stays small, and reusing the sets keeps the
// recording free of allocations.
type consumerAttrs struct {
	queueName string
	queue     colibrimonitoringbase.Attrs

	mu           sync.RWMutex
	consumedSets map[attrsKey]colibrimonitoringbase.Attrs
	rejectedSets map[attrsKey]colibrimonitoringbase.Attrs
}

type attrsKey struct {
	action  string
	outcome string
}

func newConsumerAttrs(queue string) *consumerAttrs {
	return &consumerAttrs{
		queueName:    queue,
		queue:        colibrimonitoringbase.NewAttrs(attrQueue, queue),
		consumedSets: map[attrsKey]colibrimonitoringbase.Attrs{},
		rejectedSets: map[attrsKey]colibrimonitoringbase.Attrs{},
	}
}

func (a *consumerAttrs) consumed(action, result string) colibrimonitoringbase.Attrs {
	return a.cached(a.consumedSets, attrsKey{action, result}, attrResult)
}

func (a *consumerAttrs) rejected(action, reason string) colibrimonitoringbase.Attrs {
	return a.cached(a.rejectedSets, attrsKey{action, reason}, attrReason)
}

func (a *consumerAttrs) cached(
	sets map[attrsKey]colibrimonitoringbase.Attrs,
	key attrsKey,
	outcomeAttr string,
) colibrimonitoringbase.Attrs {
	a.mu.RLock()
	attrs, ok := sets[key]
	a.mu.RUnlock()
	if ok {
		return attrs
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if attrs, ok = sets[key]; !ok {
		attrs = colibrimonitoringbase.NewAttrs(attrQueue, a.queueName, attrAction, key.action, outcomeAttr, key.outcome)
		sets[key] = attrs
	}

	return attrs
}
