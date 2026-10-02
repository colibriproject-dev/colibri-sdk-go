package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupMetricsTest installs a metric recorder and creates the module instruments against
// it, the way Initialize does.
func setupMetricsTest(t *testing.T) (*fakeMessaging, *monitoringtest.Recorder) {
	t.Helper()

	f := setupMessagingTest(t)
	recorder := monitoringtest.Install(t)
	initMetrics()
	t.Cleanup(releaseMetrics)

	return f, recorder
}

// waitProcessed waits for the consumer to report n processed messages.
func waitProcessed(t *testing.T, processed <-chan struct{}, n int) {
	t.Helper()

	for i := range n {
		select {
		case <-processed:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of %d messages were processed", i, n)
		}
	}
}

func TestMessagingMetrics(t *testing.T) {
	t.Run("Should count published messages by topic and result", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		producer := NewProducer("metrics-topic")

		require.NoError(t, producer.Publish(context.Background(), "created", "payload"))
		f.closed.Store(true)
		require.Error(t, producer.Publish(context.Background(), "created", "payload"))

		published := recorder.Metric(t, monitoring.MetricMessagingPublished)
		monitoringtest.AssertCataloged(t, published)
		assert.Equal(t, int64(1), monitoringtest.CounterValue(t, published, attrTopic, "metrics-topic", attrResult, resultSuccess))
		assert.Equal(t, int64(1), monitoringtest.CounterValue(t, published, attrTopic, "metrics-topic", attrResult, resultError))
	})

	t.Run("Should count consumed messages and their duration by queue, action and result", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		processed := make(chan struct{}, 3)

		c := startFakeConsumer(t, "metrics-queue", func(_ context.Context, msg *ProviderMessage) error {
			defer func() { processed <- struct{}{} }()

			switch msg.Action {
			case "fail":
				return errors.New("processing failed")
			case "boom":
				panic("consumer exploded")
			default:
				return nil
			}
		})

		f.ch <- NewConsumerMessage("ok", nil, nil, nil)
		f.ch <- NewConsumerMessage("fail", nil, nil, nil)
		f.ch <- NewConsumerMessage("boom", nil, nil, nil)
		waitProcessed(t, processed, 3)
		closeWithin(t, c, 5*time.Second)

		consumed := recorder.Metric(t, monitoring.MetricMessagingConsumed)
		monitoringtest.AssertCataloged(t, consumed)
		for action, result := range map[string]string{"ok": resultSuccess, "fail": resultError, "boom": resultPanic} {
			assert.Equal(t, int64(1), monitoringtest.CounterValue(t, consumed,
				attrQueue, "metrics-queue", attrAction, action, attrResult, result), action)
		}

		duration := recorder.Metric(t, monitoring.MetricMessagingProcessDuration)
		monitoringtest.AssertCataloged(t, duration)
		count, _ := monitoringtest.HistogramCount(t, duration,
			attrQueue, "metrics-queue", attrAction, "ok", attrResult, resultSuccess)
		assert.Equal(t, uint64(1), count)
	})

	t.Run("Should record the consume lag by queue and action", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		processed := make(chan struct{}, 3)

		c := startFakeConsumer(t, "lag-queue", func(context.Context, *ProviderMessage) error {
			processed <- struct{}{}
			return nil
		})

		waited := NewConsumerMessage("waited", nil, nil, nil)
		waited.setPublishTime(time.Now().Add(-2 * time.Second))
		skewed := NewConsumerMessage("skewed", nil, nil, nil)
		skewed.setPublishTime(time.Now().Add(time.Hour))
		f.ch <- waited
		f.ch <- skewed
		f.ch <- NewConsumerMessage("untimed", nil, nil, nil)
		waitProcessed(t, processed, 3)
		closeWithin(t, c, 5*time.Second)

		lag := recorder.Metric(t, monitoring.MetricMessagingConsumeLag)
		monitoringtest.AssertCataloged(t, lag)
		monitoringtest.AssertShape(t, lag, "s", attrQueue, attrAction)

		count, sum := monitoringtest.HistogramCount(t, lag, attrQueue, "lag-queue", attrAction, "waited")
		assert.Equal(t, uint64(1), count)
		assert.GreaterOrEqual(t, sum, 2.0)

		count, sum = monitoringtest.HistogramCount(t, lag, attrQueue, "lag-queue", attrAction, "skewed")
		assert.Equal(t, uint64(1), count)
		assert.Zero(t, sum, "a publish time ahead of the consumer clock is recorded as no lag")

		assert.Len(t, lag.Data.(metricdata.Histogram[float64]).DataPoints, 2,
			"a message without a publish time records no lag")
	})

	t.Run("Should count rejected messages by queue, action and reason", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		processed := make(chan struct{}, 2)

		c := startFakeConsumer(t, "rejected-queue", func(_ context.Context, msg *ProviderMessage) error {
			defer func() { processed <- struct{}{} }()

			if msg.Action == "boom" {
				panic("consumer exploded")
			}
			return errors.New("processing failed")
		})

		f.ch <- NewConsumerMessage("fail", nil, nil, nil)
		f.ch <- NewConsumerMessage("boom", nil, nil, nil)
		waitProcessed(t, processed, 2)
		closeWithin(t, c, 5*time.Second)

		rejected := recorder.Metric(t, monitoring.MetricMessagingRejected)
		monitoringtest.AssertCataloged(t, rejected)
		assert.Equal(t, int64(1), monitoringtest.CounterValue(t, rejected,
			attrQueue, "rejected-queue", attrAction, "fail", attrReason, resultError))
		assert.Equal(t, int64(1), monitoringtest.CounterValue(t, rejected,
			attrQueue, "rejected-queue", attrAction, "boom", attrReason, resultPanic))
	})

	t.Run("Should not count a successful message as rejected", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		processed := make(chan struct{}, 1)

		c := startFakeConsumer(t, "accepted-queue", func(context.Context, *ProviderMessage) error {
			processed <- struct{}{}
			return nil
		})

		f.ch <- NewConsumerMessage("ok", nil, nil, nil)
		waitProcessed(t, processed, 1)
		closeWithin(t, c, 5*time.Second)

		assert.NotContains(t, recorder.Collect(t), monitoring.MetricMessagingRejected)
	})

	t.Run("Should report the messages in flight per queue", func(t *testing.T) {
		f, recorder := setupMetricsTest(t)
		started := make(chan struct{})
		release := make(chan struct{})
		processed := make(chan struct{}, 1)

		c := startFakeConsumer(t, "in-flight-metrics-queue", func(context.Context, *ProviderMessage) error {
			close(started)
			<-release
			processed <- struct{}{}
			return nil
		})

		idle := recorder.Metric(t, monitoring.MetricMessagingInFlight)
		monitoringtest.AssertCataloged(t, idle)
		assert.Zero(t, monitoringtest.GaugeValue(t, idle, attrQueue, "in-flight-metrics-queue"))

		f.ch <- NewConsumerMessage("slow", nil, nil, nil)
		<-started
		busy := recorder.Metric(t, monitoring.MetricMessagingInFlight)
		assert.InDelta(t, 1.0, monitoringtest.GaugeValue(t, busy, attrQueue, "in-flight-metrics-queue"), 1e-9)

		close(release)
		waitProcessed(t, processed, 1)
		// the gauge drops once processMessage returns, just after the handler signaled
		assert.Eventually(t, func() bool {
			m := recorder.Metric(t, monitoring.MetricMessagingInFlight)
			return monitoringtest.GaugeValue(t, m, attrQueue, "in-flight-metrics-queue") == 0
		}, time.Second, 10*time.Millisecond)

		closeWithin(t, c, 5*time.Second)
	})

	t.Run("Should stop reporting in-flight messages once the module is closed", func(t *testing.T) {
		_, recorder := setupMetricsTest(t)
		c := startFakeConsumer(t, "released-queue", func(context.Context, *ProviderMessage) error { return nil })
		require.Contains(t, recorder.Collect(t), monitoring.MetricMessagingInFlight)

		releaseMetrics()

		assert.NotContains(t, recorder.Collect(t), monitoring.MetricMessagingInFlight)
		closeWithin(t, c, 5*time.Second)
	})
}

func TestConsumerAttrsCache(t *testing.T) {
	t.Run("Should reuse the attribute set of a repeated action and outcome", func(t *testing.T) {
		attrs := newConsumerAttrs("cache-queue")

		first := attrs.consumed("created", resultSuccess)
		second := attrs.consumed("created", resultSuccess)

		assert.Equal(t, first, second)
		assert.NotEqual(t, first, attrs.consumed("created", resultError))
		assert.NotEqual(t, first, attrs.rejected("created", resultSuccess))
	})

	t.Run("Should not allocate when recording with a cached attribute set", func(t *testing.T) {
		attrs := newConsumerAttrs("alloc-queue")
		attrs.consumed("created", resultSuccess)

		allocs := testing.AllocsPerRun(100, func() {
			attrs.consumed("created", resultSuccess)
		})

		assert.Zero(t, allocs)
	})
}
