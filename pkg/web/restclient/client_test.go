package restclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/net"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/database/cacheDB"
	"github.com/stretchr/testify/assert"
)

type userResponseTestStruct struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userResponseErrorTestStruct struct {
	Message string `json:"message"`
}

type userRequestTestStruct struct {
	ID    uint   `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type tokenResponseTestStruct struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	ExpiresIn uint64 `json:"expiresIn"`
}

var (
	ctx        = context.Background()
	wiremock   *test.WiremockContainer
	restClient *RestClient
)

func TestMain(m *testing.M) {
	monitoring.Initialize()
	test.InitializeCacheDBTest()
	wiremock = test.UseWiremockContainer(context.Background(), test.MountAbsolutPath(test.WIREMOCK_ENVIRONMENT_PATH))
	restClient = NewRestClient(&RestClientConfig{
		Name:    "test-rest-client",
		BaseURL: fmt.Sprintf("http://localhost:%d/users-api/v1", wiremock.Port()),
		Timeout: 1,
	})

	m.Run()
}

func TestGet(t *testing.T) {
	t.Run("Should return 500 status code (Internal Server Error) and nil body when timeout occurs", func(t *testing.T) {
		t.Parallel()
		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodGet,
			Path:       "/2",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.Errorf(t, response.Error(), "expected timeout error: %v\n", response.Error())
		assert.Truef(t, net.IsTimeout(response.Error()), "expected timeout error: %v\n", response.Error())
	})

	t.Run("Should return 200 status code (OK) and not nil body with list", func(t *testing.T) {
		t.Parallel()
		expected := []userResponseTestStruct{
			{ID: 1, Name: "Jaya Ganaka 1", Email: "ganaka_1_jaya@ryan.biz"},
			{ID: 2, Name: "Jaya Ganaka 2", Email: "ganaka_2_jaya@ryan.biz"},
			{ID: 3, Name: "Jaya Ganaka 3", Email: "ganaka_3_jaya@ryan.biz"},
			{ID: 4, Name: "Jaya Ganaka 4", Email: "ganaka_4_jaya@ryan.biz"},
			{ID: 5, Name: "Jaya Ganaka 5", Email: "ganaka_5_jaya@ryan.biz"},
		}

		response := Request[[]userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodGet,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.Equal(t, 5, len(*response.SuccessBody()))
		assert.EqualValues(t, expected, *response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 200 status code (OK) and not nil body with object", func(t *testing.T) {
		t.Parallel()
		expected := &userResponseTestStruct{ID: 1, Name: "Jaya Ganaka MD", Email: "ganaka_md_jaya@ryan.biz"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodGet,
			Path:       "/1",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.EqualValues(t, expected, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestPost(t *testing.T) {
	t.Run("Should return 201 status code (Created) and not nil body with object", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "User 10", Email: "user_10@email.com"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusCreated, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, uint(10), response.SuccessBody().ID)
		assert.Equal(t, newUser.Name, response.SuccessBody().Name)
		assert.Equal(t, newUser.Email, response.SuccessBody().Email)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 201 status code (Created) and nil body", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "User 100", Email: "user_100@email.com"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusCreated, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestPostWithMultipart(t *testing.T) {
	t.Run("Should return 201 status code (Created) and nil body to valid MultipartFields containing a file with custom content type", func(t *testing.T) {
		t.Parallel()
		var UploadFile io.Reader = bytes.NewBufferString("test")

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/upload",
			MultipartFields: map[string]any{
				"myfile": MultipartFile{
					FileName:    "test.txt",
					File:        UploadFile,
					ContentType: "text/plain",
				},
			},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusCreated, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 201 status code (Created) and nil body to valid MultipartFields containing a file with default content type", func(t *testing.T) {
		t.Parallel()
		var UploadFile io.Reader = bytes.NewBufferString("test")

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/upload",
			MultipartFields: map[string]any{
				"file": MultipartFile{
					FileName: "test.txt",
					File:     UploadFile,
				},
			},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusCreated, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 201 status code (Created) and nil body to valid MultipartFields containing text fields", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "User 100", Email: "user_100@email.com"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			MultipartFields: map[string]any{
				"name":  "User 100",
				"email": "user_100@email.com",
			},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusCreated, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, uint(10), response.SuccessBody().ID)
		assert.Equal(t, newUser.Name, response.SuccessBody().Name)
		assert.Equal(t, newUser.Email, response.SuccessBody().Email)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 500 status code (Internal Server Error) when send invalid MultipartFields type", func(t *testing.T) {
		t.Parallel()

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/upload",
			MultipartFields: map[string]any{
				"file": -1,
			},
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.ErrorContains(t, response.Error(), "error while sending the multipart/form-data: data type not allowed")
	})

}

func TestPut(t *testing.T) {
	t.Run("Should return 200 status code (OK) and not nil body with object", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{ID: 10, Name: "User 10 edited", Email: "user_10@email.com"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPut,
			Path:       fmt.Sprintf("/users/%d", newUser.ID),
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NoError(t, response.Error())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, newUser.ID, response.SuccessBody().ID)
		assert.Equal(t, newUser.Name, response.SuccessBody().Name)
		assert.Equal(t, newUser.Email, response.SuccessBody().Email)
	})

	t.Run("Should return 204 status code (No Content) and nil body", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{ID: 100, Name: "User 100 edited", Email: "user_100@email.com"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPut,
			Path:       fmt.Sprintf("/users/%d", newUser.ID),
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusNoContent, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestPatch(t *testing.T) {
	t.Run("Should return 200 status code (OK) and not nil body with object", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "User 10 edited"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPatch,
			Path:       "/users/10",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, uint(10), response.SuccessBody().ID)
		assert.Equal(t, newUser.Name, response.SuccessBody().Name)
		assert.NotNil(t, response.SuccessBody().Email)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 204 status code (No Content) and nil body", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "User 100 edited"}

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPatch,
			Path:       "/users/100",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusNoContent, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestDelete(t *testing.T) {
	t.Run("Should return 200 status code (OK) and not nil body with object", func(t *testing.T) {
		t.Parallel()

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodDelete,
			Path:       "/users/11",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, uint(11), response.SuccessBody().ID)
		assert.Equal(t, "User 11 deleted", response.SuccessBody().Name)
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})

	t.Run("Should return 204 status code (No Content) and nil body", func(t *testing.T) {
		t.Parallel()

		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodDelete,
			Path:       "/users/12",
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusNoContent, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.NoError(t, response.Error())
	})
}

func TestPostNotEmptyResponseBodyError(t *testing.T) {
	t.Run("Should return 500 status code (Internal Server Error) and nil body", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "Empty Body Response", Email: "post_user_empty_body@error.com"}

		response := Request[any, userResponseErrorTestStruct]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.Errorf(t, response.Error(), errResponseWithEmptyBody, http.StatusInternalServerError)
	})

	t.Run("Should return 500 status code (Internal Server Error) and not nil body with object", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "Body Response", Email: "post_user_with_body@error.com"}

		response := Request[userResponseTestStruct, userResponseErrorTestStruct]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.EqualValues(t, "Error message post user", response.ErrorBody().Message)
		assert.EqualError(t, response.Error(), "error body decoded with 500 status code")
	})

	t.Run("Should return 500 status code (Internal Server Error) when decode response error occurs", func(t *testing.T) {
		t.Parallel()
		newUser := userRequestTestStruct{Name: "Body Response", Email: "post_user_decode_error@error.com"}

		response := Request[userResponseTestStruct, userResponseErrorTestStruct]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
		assert.Nil(t, response.ErrorBody())
		assert.ErrorContains(t, response.Error(), "500")
		assert.ErrorContains(t, response.Error(), "could not decode response")
	})
}

func TestPostWithBodyString(t *testing.T) {
	t.Run("Should return 200 status code (OK) and not nil body with object when post with x-www-form-urlencoded", func(t *testing.T) {
		data := url.Values{}
		data.Set("user", "darth_vader")
		data.Set("pass", "force")
		response := Request[userResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     restClient,
			HttpMethod: http.MethodPost,
			Path:       "/login",
			Headers:    map[string]string{"Content-type": "application/x-www-form-urlencoded"},
			Body:       data.Encode(),
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusOK, response.StatusCode())
		assert.NotNil(t, response.SuccessBody())
		assert.Equal(t, uint(100), response.SuccessBody().ID)
		assert.Equal(t, "user_100@email.com", response.SuccessBody().Email)
		assert.Equal(t, "User 100", response.SuccessBody().Name)
		assert.Nil(t, response.ErrorBody())
		assert.Nil(t, response.Error())
	})
}

func TestPostWithRetry(t *testing.T) {
	retryClient := NewRestClient(&RestClientConfig{
		Name:                "test-post-with-retry-in-rest-client",
		BaseURL:             fmt.Sprintf("http://localhost:%d/users-api/v1", wiremock.Port()),
		Timeout:             100,
		Retries:             3,
		RetrySleepInSeconds: 1,
	})

	t.Run("Should return 200 status code (OK) and not nil body with object when post with x-www-form-urlencoded", func(t *testing.T) {
		newUser := userRequestTestStruct{Name: "Empty Body Response", Email: "post_user_empty_body@error.com"}

		response := Request[any, any]{
			Ctx:        ctx,
			Client:     retryClient,
			HttpMethod: http.MethodPost,
			Path:       "/users",
			Body:       &newUser,
		}.Call()

		assert.NotNil(t, response)
		assert.EqualValues(t, http.StatusInternalServerError, response.StatusCode())
		assert.Nil(t, response.SuccessBody())
	})
}

func TestPostWithCache(t *testing.T) {
	cacheDB.Initialize()
	retryClient := NewRestClient(&RestClientConfig{
		Name:                "test-post-with-cache-in-rest-client",
		BaseURL:             fmt.Sprintf("http://localhost:%d/token", wiremock.Port()),
		Timeout:             100,
		Retries:             3,
		RetrySleepInSeconds: 1,
	})
	userToken := &tokenResponseTestStruct{Type: "Bearer", Token: "1A2B3C4D5E", ExpiresIn: 123456789}
	userBody := &userRequestTestStruct{ID: 10, Name: "User 10", Email: "user_10@email.com"}
	tokenCache := cacheDB.NewCache[tokenResponseTestStruct]("user-token-cache", time.Hour)

	t.Run("Should set and return object in cache database", func(t *testing.T) {
		// Check if cache is empty
		emptyCache, err := tokenCache.One(ctx)
		assert.NoError(t, err)
		assert.Nil(t, emptyCache)

		// First call, get in api and set in cache
		request := Request[tokenResponseTestStruct, any]{
			Ctx:        ctx,
			Client:     retryClient,
			HttpMethod: http.MethodPost,
			Cache:      tokenCache,
			Body:       userBody,
		}

		firstResponse := request.Call()

		assert.NotNil(t, firstResponse)
		assert.EqualValues(t, http.StatusOK, firstResponse.StatusCode())
		assert.EqualValues(t, userToken, firstResponse.SuccessBody())
		assert.Nil(t, firstResponse.ErrorBody())
		assert.Nil(t, firstResponse.Error())

		// Check if token saved in cache
		cachedToken, err := tokenCache.One(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, cachedToken)
		assert.EqualValues(t, userToken, cachedToken)

		// Second call, get in cache and return not modified status code
		cachedResponse := request.Call()

		assert.NotNil(t, cachedResponse)
		assert.EqualValues(t, http.StatusNotModified, cachedResponse.StatusCode())
		assert.EqualValues(t, userToken, cachedResponse.SuccessBody())
		assert.Nil(t, cachedResponse.ErrorBody())
		assert.Nil(t, cachedResponse.Error())
	})
}
