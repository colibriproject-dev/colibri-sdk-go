package restserver

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	DefaultTestUserId   = "5e859dae-c879-11eb-b8bc-0242ac130003"
	DefaultTestTenantId = "5e859dae-c879-11eb-b8bc-0242ac130004"
)

// RequestTest is a contract to test http requests
type RequestTest struct {
	// Method http method
	Method string
	// Url to call in test
	Url string
	// Path to register endpoint
	Path string
	// Request headers
	Headers map[string]string
	// Body request
	Body string
	// UploadFile file to upload in multipart form file
	UploadFile *os.File
	// FormFileField field name
	FormFileField string
	// FormFileName file name
	FormFileName string
}

// NewRequestTest returns a TestResponse with result of test execution
func NewRequestTest(request *RequestTest, handlerFn func(ctx WebContext)) *TestResponse {
	app := fiber.New()
	app.Use(authenticationContextFiberMiddleware())

	path := convertUriToFiberUri(request.Path)
	app.Add([]string{request.Method}, path, func(ctx fiber.Ctx) error {
		webContext := newFiberWebContext(ctx)

		handlerFn(webContext)

		return nil
	})

	var req *http.Request

	if request.UploadFile != nil {
		b := new(bytes.Buffer)
		writer := multipart.NewWriter(b)
		formFile, err := writer.CreateFormFile(request.FormFileField, request.FormFileName)
		if err != nil {
			panic(err)
		}

		_, _ = io.Copy(formFile, request.UploadFile)
		if err = writer.Close(); err != nil {
			panic(err)
		}

		req = httptest.NewRequest(request.Method, request.Url, b)
		req.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
	} else {
		req = httptest.NewRequest(request.Method, request.Url, bytes.NewBuffer([]byte(request.Body)))
	}

	req.Header.Add(tenantIDHeader, DefaultTestTenantId)
	req.Header.Add(userIDHeader, DefaultTestUserId)
	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		panic(err)
	}

	return &TestResponse{resp: resp}
}

func convertUriToFiberUri(uri string) string {
	replacer := strings.NewReplacer(
		"{", "",
		"}", "",
	)

	paths := strings.Split(uri, "/")
	for idx, path := range paths {
		if pathIsPathParam(path) {
			paths[idx] = fmt.Sprintf(":%s", replacer.Replace(path))
		}
	}

	return strings.Join(paths, "/")
}

func pathIsPathParam(path string) bool {
	return strings.Contains(path, "{")
}
