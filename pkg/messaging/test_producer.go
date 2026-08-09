package messaging

import (
	"context"
	"errors"
	"time"
)

var ErrTestProducerTimeout = errors.New("timeout")

type testProducerConsumer struct {
	fn        func(ctx context.Context, n *ProviderMessage) error
	queueName string
}

func (c *testProducerConsumer) Consume(ctx context.Context, providerMessage *ProviderMessage) error {
	return c.fn(ctx, providerMessage)
}

func (c *testProducerConsumer) QueueName() string {
	return c.queueName
}

// TestProducer is a contract to test messaging producer
type TestProducer[T any] struct {
	producerFn func() error
	testQueue  string
	timeout    time.Duration
}

// NewTestProducer returns a pointer of TestProducer
func NewTestProducer[T any](producerFn func() error, testQueue string, timeoutInSeconds uint8) *TestProducer[T] {
	if timeoutInSeconds == 0 {
		timeoutInSeconds = 3
	}

	return &TestProducer[T]{
		producerFn: producerFn,
		testQueue:  testQueue,
		timeout:    time.Duration(timeoutInSeconds) * time.Second,
	}
}

// Execute returns a T pointer or error in test execution
func (p *TestProducer[T]) Execute() (response *T, err error) {
	// buffered so the handler never blocks on a send nobody is left to receive. Once Execute
	// gives up on the timeout it stops reading, and an unbuffered send would park the handler
	// forever, keeping its consumer from ever finishing and the shutdown from ever draining
	chSuccess := make(chan *T, 1)
	chError := make(chan error, 1)

	consumer, consumerErr := NewConsumerWithError(&testProducerConsumer{
		fn: func(ctx context.Context, providerMessage *ProviderMessage) error {
			var model T
			if err := providerMessage.DecodeMessage(&model); err != nil {
				chError <- err
				return err
			}

			chSuccess <- &model
			return nil
		},
		queueName: p.testQueue,
	})
	if consumerErr != nil {
		return nil, consumerErr
	}

	// the consumer serves this single execution: without closing it, every call would leave a
	// listener behind and the shutdown would have one more task to drain on every Execute
	defer consumer.Close()

	if err := p.producerFn(); err != nil {
		return nil, err
	}

	select {
	case response = <-chSuccess:
		return
	case err = <-chError:
		return
	case <-time.After(p.timeout):
		return nil, ErrTestProducerTimeout
	}
}
