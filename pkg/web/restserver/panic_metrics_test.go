package restserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicRecoverMetrics(t *testing.T) {
	t.Run("Should count a recovered panic by method and route template", func(t *testing.T) {
		recorder := monitoringtest.Install(t)

		app := fiber.New()
		app.Use(panicRecoverMiddleware())
		app.Get("/users/:id", func(c fiber.Ctx) error {
			// the route handler registered by injectRoutes sets the template before the user code runs
			c.Set(parameterizedURLHeaderKey, "/users/:id")
			panic("handler exploded")
		})

		for _, id := range []string{"1", "2"} {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/users/"+id, nil))
			require.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		}

		panics := recorder.Metric(t, metricPanicRecovered)
		monitoringtest.AssertShape(t, panics, "{panic}", attrHTTPRequestMethod, attrHTTPRoute)
		assert.Equal(t, int64(2), monitoringtest.CounterValue(t, panics,
			attrHTTPRequestMethod, http.MethodGet, attrHTTPRoute, "/users/:id"))
	})

	t.Run("Should fall back to the matched route when the template was not set", func(t *testing.T) {
		recorder := monitoringtest.Install(t)

		app := fiber.New()
		app.Use(panicRecoverMiddleware())
		app.Use(func(fiber.Ctx) error { panic("middleware exploded") })

		resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/orders/42", nil))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		panics := recorder.Metric(t, metricPanicRecovered)
		monitoringtest.AssertShape(t, panics, "{panic}", attrHTTPRequestMethod, attrHTTPRoute)
		assert.Equal(t, int64(1), monitoringtest.CounterValue(t, panics,
			attrHTTPRequestMethod, http.MethodPost, attrHTTPRoute, "/"),
			"the path must never be used as the route: it is unbounded")
	})

	t.Run("Should not count a request that did not panic", func(t *testing.T) {
		recorder := monitoringtest.Install(t)

		app := fiber.New()
		app.Use(panicRecoverMiddleware())
		app.Get("/ok", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.NotContains(t, recorder.Collect(t), metricPanicRecovered)
	})
}
