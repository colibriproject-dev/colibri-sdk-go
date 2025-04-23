package restserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/security"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/test"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/web/restclient"
	"github.com/stretchr/testify/assert"
)

const (
	UserId   = "6e859dae-c879-11eb-b8bc-0242ac130004"
	TenantId = "6e859dae-c879-11eb-b8bc-0242ac130005"
)

type customMiddlewareTest struct {
}

func (t *customMiddlewareTest) Apply(ctx WebContext) *MiddlewareError {
	if strings.HasPrefix(ctx.Path(), "/public/middleware/conflict") {
		return NewMiddlewareError(http.StatusConflict, errors.New("has conflict"))
	}
	return nil
}

type customAuthenticationContextMiddleware struct {
}

func (m *customAuthenticationContextMiddleware) Apply(ctx WebContext) (*security.AuthenticationContext, error) {
	if !strings.Contains(ctx.Path(), string(AuthenticatedApi)) {
		return nil, nil
	}

	authHeaders := ctx.RequestHeaders()["Authorization"]
	if len(authHeaders) < 1 || authHeaders[0] == "" {
		return nil, errUserUnauthenticated
	}
	return security.NewAuthenticationContext("ab123", "123abc"), nil

}

func beforeEnterApply(ctx WebContext) *MiddlewareError {
	if strings.HasPrefix(ctx.Path(), "/public/hook/conflict") {
		return NewMiddlewareError(http.StatusConflict, errors.New("has hook conflict"))
	}
	return nil
}

func TestStartRestServer(t *testing.T) {
	ctx := context.Background()
	test.InitializeBaseTest()

	type Resp struct {
		Msg string `json:"msg" validate:"required"`
	}

	type Query struct {
		Msg  string `form:"msg"`
		Size uint8  `form:"size"`
	}

	listener := func() (l net.Listener) {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Error("could not get available port")
		}

		return l
	}

	AddRoutes([]Route{
		{
			URI:    "test-public-endpoint",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "test-public-endpoint"})
			},
			Prefix: PublicApi,
		},
		{
			URI:    "test-public-endpoint",
			Method: http.MethodPost,
			Prefix: PublicApi,
			Function: func(ctx WebContext) {
				body, err := ctx.StringBody()
				if err != nil {
					ctx.ErrorResponse(http.StatusInternalServerError, err)
					return
				}
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: body})
			},
		},
		{
			URI:    "test-private-endpoint/{msg}",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				msg := ctx.PathParam("msg")
				ctx.JsonResponse(http.StatusOK, &Resp{msg})
			},
			Prefix: PrivateApi,
		},
		{
			URI:    "test-authenticated-endpoint",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, ctx.AuthenticationContext())
			},
			Prefix: AuthenticatedApi,
		},
		{
			URI:    "test-authenticated-endpoint/{msg}",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				msg := ctx.PathParam("msg")
				ctx.JsonResponse(http.StatusOK, &Resp{msg})
			},
			Prefix: AuthenticatedApi,
		},
		{
			URI:    "test-no-prefix-endpoint",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "test-no-prefix-endpoint"})
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-empty-body",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.EmptyResponse(http.StatusNoContent)
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-error-body",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.ErrorResponse(http.StatusInternalServerError, fmt.Errorf("test-error-body"))
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-request-header",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, ctx.RequestHeader("X-Id"))
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-request-headers",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, ctx.RequestHeaders())
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-query-param",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, ctx.QueryParam("size"))
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-query-array-param",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, ctx.QueryArrayParam("idList"))
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-decode-query-params",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				var result Query
				if err := ctx.DecodeQueryParams(&result); err != nil {
					ctx.ErrorResponse(http.StatusBadRequest, err)
				}
				ctx.JsonResponse(http.StatusOK, result)
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-decode-body",
			Method: http.MethodPost,
			Function: func(ctx WebContext) {
				var result Resp
				if err := ctx.DecodeBody(&result); err != nil {
					ctx.ErrorResponse(http.StatusBadRequest, err)
					return
				}
				ctx.JsonResponse(http.StatusOK, result)
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-context",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				msg := fmt.Sprintf("%s", ctx.Context())
				ctx.AddHeader("context", msg)
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: msg})
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "test-serve-file",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				path := test.MountAbsolutPath(test.REST_ENVIRONMENT_PATH) + "/resp.json"
				ctx.AddHeaders(map[string]string{"path": path})
				ctx.ServeFile(path)
			},
			Prefix: NoPrefix,
		},
		{
			URI:    "middleware/conflict",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "conflict"})
			},
			Prefix: PublicApi,
		},
		{
			URI:    "middleware/success",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "success"})
			},
			Prefix: PublicApi,
		},
		{
			URI:    "hook/conflict",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "conflict"})
			},
			Prefix:      PublicApi,
			BeforeEnter: beforeEnterApply,
		},
		{
			URI:    "hook/success",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "success"})
			},
			Prefix:      PublicApi,
			BeforeEnter: beforeEnterApply,
		},
	})

	l := listener()
	config.PORT = l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	Use(&customMiddlewareTest{})
	go ListenAndServe()
	time.Sleep(1 * time.Second)

	baseURL := fmt.Sprintf("http://localhost:%d", config.PORT)
	client := restclient.NewRestClient(&restclient.RestClientConfig{
		Name:    "test-server",
		BaseURL: baseURL,
		Timeout: 100,
	})

	t.Run("Should return status 200 (OK) in health-check", func(t *testing.T) {
		response := restclient.Request[healtCheck, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/health",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.Equal(t, "OK", response.SuccessBody().Status)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return error 404 (not found) when endpoint not exists", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/not-exists-endpoint",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusNotFound, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.Error(t, response.Error(), "404 staus code")
	})

	t.Run("Should return 200 (OK) in public api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/public/test-public-endpoint",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, "test-public-endpoint", response.SuccessBody().Msg)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 200 (OK) in private api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/private/test-private-endpoint/abc",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, "abc", response.SuccessBody().Msg)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return error 401 (Unauthorized) when not inform the userId in authenticated api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/test-authenticated-endpoint/abc",
			Headers:    map[string]string{tenantIDHeader: TenantId},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusUnauthorized, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.Error(t, errors.New("401 statusCode"), response.Error())
	})

	t.Run("Should return error 401 (Unauthorized) when not inform the tenantId in authenticated api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/test-authenticated-endpoint/abc",
			Headers:    map[string]string{userIDHeader: UserId},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusUnauthorized, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.Error(t, errors.New("401 statusCode"), response.Error())
	})

	t.Run("Should return error 401 (Unauthorized) when not inform the credentials in authenticated api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/test-authenticated-endpoint/abc",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusUnauthorized, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.Error(t, errors.New("401 statusCode"), response.Error())
	})

	t.Run("Should return 200 (OK) in authenticated api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/test-authenticated-endpoint/abc",
			Headers:    map[string]string{tenantIDHeader: TenantId, userIDHeader: UserId},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, "abc", response.SuccessBody().Msg)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 200 (OK) in no prefix api", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-no-prefix-endpoint",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, "test-no-prefix-endpoint", response.SuccessBody().Msg)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return error response body", func(t *testing.T) {
		expected := &Error{"test-error-body"}

		response := restclient.Request[any, Error]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-error-body",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.EqualValues(t, expected, response.ErrorBody())
		assert.Error(t, errors.New("500 statusCode"), response.Error())
	})

	t.Run("Should return empty response body", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-empty-body",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusNoContent, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return request header", func(t *testing.T) {
		expected := &[]string{"123"}

		response := restclient.Request[[]string, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-request-header",
			Headers:    map[string]string{"X-Id": "123"},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return request headers", func(t *testing.T) {
		expected := []string{"456"}

		response := restclient.Request[map[string][]string, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-request-headers",
			Headers:    map[string]string{"X-Id": "456"},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, (*response.SuccessBody())["X-Id"])
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return query param", func(t *testing.T) {
		expected := "10"

		response := restclient.Request[string, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-query-param?size=10",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, *response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return query array param", func(t *testing.T) {
		expected := []string{"10", "20", "30"}

		response := restclient.Request[[]string, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-query-array-param?page=1&idList=10,20,30",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, *response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return decoded query param", func(t *testing.T) {
		expected := &Query{Msg: "decoded", Size: 10}

		response := restclient.Request[Query, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-decode-query-params?size=10&msg=decoded",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return bad request when an error occurred in decoded body", func(t *testing.T) {
		response := restclient.Request[Resp, Error]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodPost,
			Path:       "/test-decode-body",
			Body:       &Resp{},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusBadRequest, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.ErrorContains(t, response.Error(), "400 status code")
	})

	t.Run("Should return decoded body", func(t *testing.T) {
		expected := &Resp{Msg: "decoded"}

		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodPost,
			Path:       "/test-decode-body",
			Body:       expected,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return context", func(t *testing.T) {
		expected := &Resp{Msg: "context.Background"}

		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-context",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return server file", func(t *testing.T) {
		expected := &Resp{Msg: "test file"}

		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/test-serve-file",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("should validate custom middleware with error", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/public/middleware/conflict",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusConflict, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.ErrorContains(t, response.Error(), "409 status code")
	})

	t.Run("should validate custom middleware with success", func(t *testing.T) {
		expected := &Resp{Msg: "success"}

		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/public/middleware/success",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("should validate hook before enter route with error", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/public/hook/conflict",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusConflict, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.NotNil(t, response.ErrorBody())
		assert.ErrorContains(t, response.Error(), "409 status code")
	})

	t.Run("should validate hook before enter route with success", func(t *testing.T) {
		expected := &Resp{Msg: "success"}

		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/public/hook/success",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestStartRestServerCustomAuthMiddleware(t *testing.T) {
	ctx := context.Background()
	test.InitializeBaseTest()

	type Resp struct {
		Msg string `json:"msg" validate:"required"`
	}

	listener := func() (l net.Listener) {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Error("could not get available port")
		}
		return l
	}

	AddRoutes([]Route{
		{
			URI:    "users",
			Method: http.MethodGet,
			Function: func(ctx WebContext) {
				ctx.JsonResponse(http.StatusOK, &Resp{Msg: "test-custom-authentication-middleware"})
			},
			Prefix: AuthenticatedApi,
		},
	})

	l := listener()
	config.PORT = l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	CustomAuthMiddleware(&customAuthenticationContextMiddleware{})
	go ListenAndServe()
	time.Sleep(1 * time.Second)
	client := restclient.NewRestClient(&restclient.RestClientConfig{
		Name:    "test-server",
		BaseURL: fmt.Sprintf("http://localhost:%d", config.PORT),
		Timeout: 1,
	})

	t.Run("Should return status 200", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/users",
			Headers:    map[string]string{"Authorization": "abcd1234"},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, "test-custom-authentication-middleware", response.SuccessBody().Msg)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return error 401", func(t *testing.T) {
		response := restclient.Request[Resp, any]{
			Ctx:        ctx,
			Client:     client,
			HttpMethod: http.MethodGet,
			Path:       "/api/users",
		}.Call()

		assert.NotNil(t, response)
		assert.Error(t, response.Error(), "401 statusCode")
		assert.Nil(t, response.SuccessBody())
	})
}
