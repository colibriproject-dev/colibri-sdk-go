package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

type testProducer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestProducerTest(t *testing.T) {
	test.InitializeTestLocalstack()
	Initialize()

	ctx := context.Background()
	msg := &testProducer{ID: 1, Name: "TEST PRODUCER"}
	topicAndQueue := "COLIBRI_PROJECT_PRODUCER_TEST_APP_CONSUMER"

	t.Run("should return new testproducer", func(t *testing.T) {
		fn := func() error { return nil }
		producer := NewTestProducer[testProducer](fn, topicAndQueue, 1)

		assert.NotNil(t, producer)
		assert.NotNil(t, producer.producerFn)
		assert.Equal(t, topicAndQueue, producer.testQueue)
		assert.Equal(t, time.Duration(1000000000), producer.timeout)
	})

	t.Run("should return new testproducer with default timeout", func(t *testing.T) {
		fn := func() error { return nil }
		producer := NewTestProducer[testProducer](fn, topicAndQueue, 0)

		assert.NotNil(t, producer)
		assert.NotNil(t, producer.producerFn)
		assert.Equal(t, topicAndQueue, producer.testQueue)
		assert.Equal(t, time.Duration(3000000000), producer.timeout)
	})

	t.Run("should return success in test", func(t *testing.T) {
		resp, err := NewTestProducer[testProducer](
			func() error { return NewProducer(topicAndQueue).Publish(ctx, "TEST", msg) },
			topicAndQueue,
			10,
		).Execute()

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, msg, resp)
	})
}
