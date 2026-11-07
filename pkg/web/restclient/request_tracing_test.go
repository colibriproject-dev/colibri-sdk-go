package restclient

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kindRecorder records the kind of every transaction started, delegating everything else
// to the no-op implementation.
type kindRecorder struct {
	colibrimonitoringbase.Monitoring

	mu    sync.Mutex
	kinds map[string][]colibrimonitoringbase.SpanKind
}

func (r *kindRecorder) StartTransaction(ctx context.Context, name string, kind colibrimonitoringbase.SpanKind) (any, context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kinds[name] = append(r.kinds[name], kind)

	return nil, ctx
}

// Not parallel: it swaps the global monitoring instance.
func TestCallTransactionIsNotAClientSpan(t *testing.T) {
	t.Run("Should wrap the call in an internal span, leaving the client span to the transport", func(t *testing.T) {
		recorder := &kindRecorder{Monitoring: colibrimonitoringbase.NewOthers(), kinds: map[string][]colibrimonitoringbase.SpanKind{}}
		t.Cleanup(monitoring.ReplaceInstance(recorder))

		response := Request[userResponseTestStruct, any]{
			Ctx:        context.Background(),
			Client:     restClient,
			HttpMethod: http.MethodGet,
			Path:       "/1",
		}.Call()
		require.EqualValues(t, http.StatusOK, response.StatusCode())

		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		assert.Equal(t, []colibrimonitoringbase.SpanKind{colibrimonitoringbase.SpanKindInternal}, recorder.kinds["HTTP Client"])
	})
}
