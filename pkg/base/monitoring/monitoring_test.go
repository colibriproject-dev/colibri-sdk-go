package monitoring

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestProductionMonitoring(t *testing.T) {
	config.NEW_RELIC_LICENSE = "abcdefghijklmopqrstuvwxyz1234567890aNRAL"
	config.APP_NAME = "test"
	config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
	assert.True(t, config.IsProductionEnvironment())

	Initialize()
	assert.NotNil(t, instance)

	t.Run("Should get transaction in context", func(t *testing.T) {
		txnName := "txn-test"

		_, ctx := StartTransaction(context.Background(), txnName)
		transaction := GetTransactionInContext(ctx)
		EndTransaction(transaction)

		assert.NotNil(t, transaction)
	})

	t.Run("Should get nil when transaction not in context", func(t *testing.T) {
		transaction := GetTransactionInContext(context.Background())

		assert.Nil(t, transaction)
	})

	t.Run("Should start/end transaction, start/end segment and notice error", func(t *testing.T) {
		segName := "txn-segment-test"

		transaction, ctx := StartWebRequest(context.Background(), http.Header{}, "/", http.MethodGet)
		segment := StartTransactionSegment(ctx, segName, map[string]string{
			"TestKey": "TestValue",
		})

		EndTransactionSegment(segment)
		NoticeError(transaction, errors.New("test notice error"))
		EndTransaction(transaction)

		assert.NotNil(t, transaction)
		assert.NotNil(t, segment)
		assert.NotEmpty(t, ctx)
	})
}
