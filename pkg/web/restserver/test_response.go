package restserver

import (
	"encoding/json"
	"io"
	"net/http"
)

// TestResponse is a contract to test http requests responses
type TestResponse struct {
	resp *http.Response
}

// StatusCode returns the result status code
func (r *TestResponse) StatusCode() int {
	return r.resp.StatusCode
}

// DecodeBody returns the result body decoded
func (r *TestResponse) DecodeBody(object any) error {
	body, err := io.ReadAll(r.resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, object)
}

// RawBody returns raw body
func (r *TestResponse) RawBody() io.ReadCloser {
	return r.resp.Body
}

// StringBody return a string body
func (r *TestResponse) StringBody() (string, error) {
	b, err := io.ReadAll(r.resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), err
}

// Headers returns response headers
func (r *TestResponse) Headers() http.Header {
	return r.resp.Header
}

// Header gets header by key
func (r *TestResponse) Header(key string) string {
	return r.resp.Header.Get(key)
}
