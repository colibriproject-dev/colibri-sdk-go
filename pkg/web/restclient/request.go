package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"reflect"
	"strings"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/database/cacheDB"
)

const (
	requestCtxIsEmpty    string = "context is empty"
	requestClientIsEmpty string = "client is empty"
	requestMethodIsEmpty string = "http method is empty"
	retriesWarnMsg       string = "[%dx] call to the url '%s'. status code = %d | general error: %v | response error: %v"
)

// Request struct for http requests
type Request[T ResponseSuccessData, E ResponseErrorData] struct {
	Ctx        context.Context
	Client     *RestClient
	HttpMethod string // use http.methodXXX

	Cache           *cacheDB.Cache[T]
	Path            string
	Headers         map[string]string
	Body            any
	MultipartFields map[string]any
	writer          *multipart.Writer
}

// Call executes the HTTP request and handles retries and caching.
//
// It validates the request, checks cache, executes the request with retries, handles success, and logs errors.
// Returns the response data.
func (req Request[T, E]) Call() (response ResponseData[T, E]) {
	if err := req.validate(); err != nil {
		return newResponseData[T, E](http.StatusInternalServerError, nil, nil, nil, err)
	}

	if req.hasCache() {
		data, _ := req.Cache.One(req.Ctx)
		if data != nil {
			return newResponseData[T, E](http.StatusNotModified, nil, data, nil, nil)
		}
	}

	for execution := uint8(0); execution <= req.Client.retries; execution++ {
		response = req.execute()
		if response.HasSuccess() {
			if req.hasCache() {
				req.Cache.Set(req.Ctx, response.SuccessBody())
			}
			break
		}

		time.Sleep(req.getSleepDuration())
		if req.Client.retries != 0 {
			logging.
				Warn(req.Ctx).
				Msgf(retriesWarnMsg, execution+1, req.getUrl(), response.StatusCode(), response.Error(), response.ErrorBody())
		}
	}

	return
}

// validate checks if the Request struct fields are valid and returns an error if any are missing.
//
// No parameters.
// Returns an error.
func (rc *Request[T, E]) validate() error {
	if rc.Ctx == nil {
		return errors.New(requestCtxIsEmpty)
	}

	if rc.Client == nil {
		return errors.New(requestClientIsEmpty)
	}

	if rc.HttpMethod == "" {
		return errors.New(requestMethodIsEmpty)
	}

	return nil
}

// hasCache checks if the Request struct has a cache attached.
//
// No parameters.
// Returns a boolean.
func (rc *Request[T, E]) hasCache() bool {
	return rc.Cache != nil
}

// getUrl returns the full URL by combining the base URL with the path.
//
// No parameters.
// Returns a string.
func (rc *Request[T, E]) getUrl() string {
	return fmt.Sprintf("%s%s", rc.Client.baseURL, rc.Path)
}

// getSleepDuration returns the sleep duration as a time.Duration.
//
// No parameters.
// Returns a time.Duration.
func (rc *Request[T, E]) getSleepDuration() time.Duration {
	return time.Duration(rc.Client.retrySleep) * time.Second
}

// getBytesBody returns the body as an io.Reader and an error.
//
// No parameters.
// Returns an io.Reader and an error.
func (rc *Request[T, E]) getBytesBody() (io.Reader, error) {
	if rc.Body == nil && len(rc.MultipartFields) == 0 {
		return nil, nil
	}

	if len(rc.MultipartFields) > 0 {
		return rc.processMultipartFields()
	}

	if reflect.ValueOf(rc.Body).Kind() == reflect.String {
		return strings.NewReader(fmt.Sprintf("%v", rc.Body)), nil
	}

	requestBody, err := json.Marshal(rc.Body)
	if err != nil {
		return nil, fmt.Errorf("could not marshal body: %w", err)
	}

	return bytes.NewBuffer(requestBody), nil
}

// processMultipartFields processes the multipart fields and returns the io.Reader and an error.
//
// No parameters.
// Returns an io.Reader and an error.
func (rc *Request[T, E]) processMultipartFields() (io.Reader, error) {
	body := &bytes.Buffer{}
	rc.writer = multipart.NewWriter(body)
	for fieldName, contentField := range rc.MultipartFields {
		if err := rc.processField(fieldName, contentField); err != nil {
			return nil, err
		}
	}

	if err := rc.writer.Close(); err != nil {
		return nil, err
	}

	return body, nil
}

// processField processes the field based on its type and performs the necessary actions accordingly.
//
// fieldName: the name of the field being processed (string).
// contentField: the content of the field being processed (interface{}).
// Returns an error if any issues occur during processing.
func (rc *Request[T, E]) processField(fieldName string, contentField interface{}) error {
	if file, ok := contentField.(MultipartFile); ok {
		part, err := rc.createFilePart(fieldName, file)
		if err != nil {
			return err
		}

		if _, err := io.Copy(part, file.File); err != nil {
			return err
		}
	} else if str, ok := contentField.(string); ok {
		if err := rc.writer.WriteField(fieldName, str); err != nil {
			return err
		}
	} else {
		return errors.New("error while sending the multipart/form-data: data type not allowed")
	}

	return nil
}

// createFilePart creates a file part for the request based on the field name and the file.
//
// fieldName: the name of the field for the file part (string).
// file: the file to be processed (MultipartFile).
// Returns an io.Writer and an error.
func (rc *Request[T, E]) createFilePart(fieldName string, file MultipartFile) (io.Writer, error) {
	if file.ContentType != "" {
		return rc.createCustomContentTypeFormFile(fieldName, file)
	}

	return rc.writer.CreateFormFile(fieldName, file.FileName)
}

// createCustomContentTypeFormFile creates a custom content type form file for the request.
//
// fieldName: the name of the field for the file part (string).
// file: the file to be processed (MultipartFile).
// Returns an io.Writer and an error.
func (rc *Request[T, E]) createCustomContentTypeFormFile(fieldName string, file MultipartFile) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, file.FileName))
	h.Set("Content-Type", file.ContentType)

	return rc.writer.CreatePart(h)
}

// addHeadersInRequest sets the headers in the HTTP request based on the headers map in the Request struct.
//
// req: the HTTP request to add headers to (*http.Request).
// Return type: void
func (rc *Request[T, E]) addHeadersInRequest(req *http.Request) {
	for key, value := range rc.Headers {
		req.Header.Set(key, value)
	}
}

// execute executes the HTTP request and processes the response data.
//
// No parameters.
// Returns a ResponseData containing the response data and any errors.
func (req *Request[T, E]) execute() (response ResponseData[T, E]) {
	if !req.Client.cb.Ready() {
		return newResponseData[T, E](http.StatusInternalServerError, nil, nil, nil, errors.New(errServiceNotAvailable))
	}

	var err error
	defer func() {
		err = req.Client.cb.Done(req.Ctx, err)
	}()

	bytesBody, err := req.getBytesBody()
	if err != nil {
		return newResponseData[T, E](http.StatusInternalServerError, nil, nil, nil, err)
	}

	request, err := http.NewRequestWithContext(req.Ctx, string(req.HttpMethod), req.getUrl(), bytesBody)
	req.addHeadersInRequest(request)
	if len(req.MultipartFields) > 0 {
		request.Header.Add("Content-Type", req.writer.FormDataContentType())
	}

	resp, err := req.Client.client.Do(request)
	if err != nil {
		return newResponseData[T, E](http.StatusInternalServerError, nil, nil, nil, err)
	}

	defer resp.Body.Close()
	if !statusCodeIsOK(resp) {
		errBody, err := processErrorResponse[E](resp)
		return newResponseData[T, E](resp.StatusCode, resp.Header, nil, errBody, err)
	}

	decodedResponse, err := decodeResponse[T](resp)
	return newResponseData[T, E](resp.StatusCode, resp.Header, decodedResponse, nil, err)
}

func statusCodeIsOK(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// processErrorResponse decodes the response error data and handles different error scenarios.
//
// Takes an http.Response as input.
// Returns a pointer to the response error data and an error.
func processErrorResponse[E ResponseErrorData](resp *http.Response) (*E, error) {
	respErr, derr := decodeResponse[E](resp)
	if respErr != nil {
		return respErr, fmt.Errorf("error body decoded with %d status code", resp.StatusCode)
	} else if derr != nil {
		return nil, fmt.Errorf("%d status code. body decoder error: %w", resp.StatusCode, derr)
	}

	return nil, fmt.Errorf(errResponseWithEmptyBody, resp.StatusCode)
}

// decodeResponse decodes the response body from an http response into a generic type T.
//
// resp: the http response object to decode
// Returns a pointer to the decoded response model of type T and an error
func decodeResponse[T any](resp *http.Response) (*T, error) {
	if resp.ContentLength == 0 {
		return nil, nil
	}

	var responseModel T
	err := json.NewDecoder(resp.Body).Decode(&responseModel)
	switch {
	case err == io.EOF:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	return &responseModel, nil
}
