package restclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testSuccessBody struct {
	Message string
}

type testErrorBody struct {
	Error string
}

func TestResponseData(t *testing.T) {
	t.Run("should handle status code correctly", func(t *testing.T) {
		expectedStatusCode := http.StatusOK
		response := newResponseData[testSuccessBody, testErrorBody](
			expectedStatusCode,
			nil,
			nil,
			nil,
			nil,
		)

		actualStatusCode := response.StatusCode()
		assert.Equal(t, expectedStatusCode, actualStatusCode)
	})

	t.Run("should handle headers correctly", func(t *testing.T) {
		expectedHeaders := http.Header{
			"Content-Type": []string{"application/json"},
		}
		response := newResponseData[testSuccessBody, testErrorBody](
			http.StatusOK,
			expectedHeaders,
			nil,
			nil,
			nil,
		)

		actualHeaders := response.Headers()
		assert.Equal(t, expectedHeaders, actualHeaders)
	})

	t.Run("should handle success body correctly", func(t *testing.T) {
		expectedBody := &testSuccessBody{Message: "success"}
		response := newResponseData[testSuccessBody, testErrorBody](
			http.StatusOK,
			nil,
			expectedBody,
			nil,
			nil,
		)

		actualBody := response.SuccessBody()
		assert.Equal(t, expectedBody, actualBody)
	})

	t.Run("should handle error body correctly", func(t *testing.T) {
		expectedBody := &testErrorBody{Error: "error"}
		response := newResponseData[testSuccessBody, testErrorBody](
			http.StatusBadRequest,
			nil,
			nil,
			expectedBody,
			nil,
		)

		actualBody := response.ErrorBody()
		assert.Equal(t, expectedBody, actualBody)
	})

	t.Run("should handle error correctly", func(t *testing.T) {
		expectedError := assert.AnError
		response := newResponseData[testSuccessBody, testErrorBody](
			http.StatusInternalServerError,
			nil,
			nil,
			nil,
			expectedError,
		)

		actualError := response.Error()
		assert.Equal(t, expectedError, actualError)
	})

	t.Run("should check response types correctly", func(t *testing.T) {
		tests := []struct {
			name           string
			statusCode     int
			expectedResult bool
			checkFunc      func(ResponseData[testSuccessBody, testErrorBody]) bool
		}{
			{
				name:           "informational response",
				statusCode:     http.StatusContinue,
				expectedResult: true,
				checkFunc:      func(r ResponseData[testSuccessBody, testErrorBody]) bool { return r.IsInformationalResponse() },
			},
			{
				name:           "successful response",
				statusCode:     http.StatusOK,
				expectedResult: true,
				checkFunc:      func(r ResponseData[testSuccessBody, testErrorBody]) bool { return r.IsSuccessfulResponse() },
			},
			{
				name:           "redirection message",
				statusCode:     http.StatusMovedPermanently,
				expectedResult: true,
				checkFunc:      func(r ResponseData[testSuccessBody, testErrorBody]) bool { return r.IsRedirectionMessage() },
			},
			{
				name:           "client error response",
				statusCode:     http.StatusBadRequest,
				expectedResult: true,
				checkFunc:      func(r ResponseData[testSuccessBody, testErrorBody]) bool { return r.IsClientErrorResponse() },
			},
			{
				name:           "server error response",
				statusCode:     http.StatusInternalServerError,
				expectedResult: true,
				checkFunc:      func(r ResponseData[testSuccessBody, testErrorBody]) bool { return r.IsServerErrorResponse() },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				response := newResponseData[testSuccessBody, testErrorBody](
					tt.statusCode,
					nil,
					nil,
					nil,
					nil,
				)

				result := tt.checkFunc(response)
				assert.Equal(t, tt.expectedResult, result)
			})
		}
	})

	t.Run("should check error and success states correctly", func(t *testing.T) {
		tests := []struct {
			name            string
			statusCode      int
			errorBody       *testErrorBody
			err             error
			expectedError   bool
			expectedSuccess bool
		}{
			{
				name:            "successful response",
				statusCode:      http.StatusOK,
				errorBody:       nil,
				err:             nil,
				expectedError:   false,
				expectedSuccess: true,
			},
			{
				name:            "error with error body",
				statusCode:      http.StatusBadRequest,
				errorBody:       &testErrorBody{Error: "error"},
				err:             nil,
				expectedError:   true,
				expectedSuccess: false,
			},
			{
				name:            "error with err",
				statusCode:      http.StatusInternalServerError,
				errorBody:       nil,
				err:             assert.AnError,
				expectedError:   true,
				expectedSuccess: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				response := newResponseData[testSuccessBody, testErrorBody](
					tt.statusCode,
					nil,
					nil,
					tt.errorBody,
					tt.err,
				)

				assert.Equal(t, tt.expectedError, response.HasError())
				assert.Equal(t, tt.expectedSuccess, response.HasSuccess())
			})
		}
	})
}
