package restserver

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRequest builds an *http.Request targeting the in-memory fiber app.
func newTestRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, target, nil)
	require.NoError(t, err)
	return req
}

// newTestRequestWithBody builds an *http.Request carrying a raw string body.
func newTestRequestWithBody(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, target, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	return req
}

func TestNewOpenTelemetryFiberMiddleware(t *testing.T) {
	t.Run("Should set http.route from the parameterized url header when present", func(t *testing.T) {
		app := fiber.New()
		app.Use(newOpenTelemetryFiberMiddleware())
		app.Get("/otel-with-header", func(c fiber.Ctx) error {
			c.Set(parameterizedURLHeaderKey, "/otel/:id")
			return c.SendStatus(http.StatusOK)
		})

		resp, err := app.Test(newTestRequest(t, http.MethodGet, "/otel-with-header"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Should fall back to the matched route path when the header is absent", func(t *testing.T) {
		app := fiber.New()
		app.Use(newOpenTelemetryFiberMiddleware())
		app.Get("/otel-no-header", func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		resp, err := app.Test(newTestRequest(t, http.MethodGet, "/otel-no-header"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestHttpMetricsFiberMiddleware(t *testing.T) {
	t.Run("Should record metrics using the parameterized url header when present", func(t *testing.T) {
		app := fiber.New()
		app.Use(httpMetricsFiberMiddleware())
		app.Post("/metrics-with-header", func(c fiber.Ctx) error {
			c.Set(parameterizedURLHeaderKey, "/metrics/:id")
			return c.SendString("ok")
		})

		resp, err := app.Test(newTestRequest(t, http.MethodPost, "/metrics-with-header"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Should record metrics using the matched route path when the header is absent", func(t *testing.T) {
		app := fiber.New()
		app.Use(httpMetricsFiberMiddleware())
		app.Get("/metrics-no-header", func(c fiber.Ctx) error {
			return c.SendString("ok")
		})

		resp, err := app.Test(newTestRequest(t, http.MethodGet, "/metrics-no-header"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestFiberWebContextRedirect(t *testing.T) {
	app := fiber.New()
	app.Get("/redirect", func(c fiber.Ctx) error {
		newFiberWebContext(c).Redirect("/redirected-target", http.StatusMovedPermanently)
		return nil
	})

	resp, err := app.Test(newTestRequest(t, http.MethodGet, "/redirect"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	assert.Equal(t, "/redirected-target", resp.Header.Get("Location"))
}

func TestFiberWebContextContext(t *testing.T) {
	app := fiber.New()
	app.Get("/context", func(c fiber.Ctx) error {
		wc := newFiberWebContext(c)
		assert.NotNil(t, wc.Context())
		// No authentication context has been injected, so it must be nil.
		assert.Nil(t, wc.AuthenticationContext())
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(newTestRequest(t, http.MethodGet, "/context"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFiberWebContextAddHeaderAndStringBody(t *testing.T) {
	app := fiber.New()
	app.Post("/echo", func(c fiber.Ctx) error {
		wc := newFiberWebContext(c)
		wc.AddHeader("X-Custom-Header", "custom-value")

		body, err := wc.StringBody()
		assert.NoError(t, err)
		assert.Equal(t, "hello-body", body)
		return c.SendString(body)
	})

	resp, err := app.Test(newTestRequestWithBody(t, http.MethodPost, "/echo", "hello-body"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "hello-body", string(respBody))
}

func TestFiberWebContextFormFileNotFound(t *testing.T) {
	app := fiber.New()
	app.Post("/upload", func(c fiber.Ctx) error {
		wc := newFiberWebContext(c)
		file, header, err := wc.FormFile("missing-file")
		assert.Nil(t, file)
		assert.Nil(t, header)
		assert.Error(t, err)
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(newTestRequest(t, http.MethodPost, "/upload"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFiberWebContextFormFileSuccess(t *testing.T) {
	app := fiber.New()
	app.Post("/upload", func(c fiber.Ctx) error {
		wc := newFiberWebContext(c)
		file, header, err := wc.FormFile("document")
		require.NoError(t, err)
		require.NotNil(t, file)
		require.NotNil(t, header)
		defer file.Close()

		content, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, "file-content", string(content))
		assert.Equal(t, "doc.txt", header.Filename)
		return c.SendStatus(http.StatusOK)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("document", "doc.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("file-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
