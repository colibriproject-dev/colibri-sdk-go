package restserver

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/validator"
	"github.com/gofiber/fiber/v2"
)

type fiberWebContext struct {
	ctx *fiber.Ctx
}

func newFiberWebContext(ctx *fiber.Ctx) *fiberWebContext {
	return &fiberWebContext{ctx: ctx}
}

func (f *fiberWebContext) Context() context.Context {
	return f.ctx.UserContext()
}

func (f *fiberWebContext) AuthenticationContext() *security.AuthenticationContext {
	return security.GetAuthenticationContext(f.Context())
}

func (f *fiberWebContext) RequestHeader(key string) []string {
	return []string{f.ctx.Get(key, "")}
}

func (f *fiberWebContext) RequestHeaders() map[string][]string {
	headers := make(map[string][]string)

	f.ctx.Context().Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = strings.Split(string(value), ";")
	})

	return headers
}

func (f *fiberWebContext) PathParam(key string) string {
	return f.ctx.Params(key)
}

func (f *fiberWebContext) QueryParam(key string) string {
	return f.ctx.Query(key)
}

func (f *fiberWebContext) QueryArrayParam(key string) []string {
	return strings.Split(f.ctx.Query(key), ",")
}

func (f *fiberWebContext) DecodeQueryParams(value any) error {
	queryParams := make(map[string][]string)

	f.ctx.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = strings.Split(string(value), ",")
	})

	if err := validator.FormDecode(value, queryParams); err != nil {
		return err
	}

	return validator.Struct(value)
}

func (f *fiberWebContext) DecodeBody(value any) error {
	if err := json.Unmarshal(f.ctx.Body(), value); err != nil {
		return err
	}

	return validator.Struct(value)
}

func (f *fiberWebContext) DecodeFormData(value any) error {
	formData := make(map[string][]string)

	f.ctx.Request().PostArgs().VisitAll(func(key, val []byte) {
		keyStr := string(key)
		valStr := string(val)
		if existing, ok := formData[keyStr]; ok {
			formData[keyStr] = append(existing, valStr)
		} else {
			formData[keyStr] = []string{valStr}
		}
	})

	if err := validator.FormDecode(value, formData); err != nil {
		return err
	}

	return validator.Struct(value)
}

func (f *fiberWebContext) AddHeader(key string, value string) {
	f.ctx.Response().Header.Add(key, value)
}

func (f *fiberWebContext) AddHeaders(headers map[string]string) {
	for key, value := range headers {
		f.ctx.Response().Header.Add(key, value)
	}
}

func (f *fiberWebContext) ServeFile(path string) {
	if err := f.ctx.SendFile(path); err != nil {
		f.ErrorResponse(http.StatusInternalServerError, err)
	}
}

func (f *fiberWebContext) JsonResponse(statusCode int, body any) {
	f.ctx.Status(statusCode)
	if err := f.ctx.JSON(body); err != nil {
		f.ErrorResponse(http.StatusInternalServerError, err)
	}
}

func (f *fiberWebContext) ErrorResponse(statusCode int, err error) {
	f.JsonResponse(statusCode, Error{err.Error()})
}

func (f *fiberWebContext) EmptyResponse(statusCode int) {
	f.ctx.Response().SetStatusCode(statusCode)
}

func (f *fiberWebContext) Redirect(url string, statusCode int) {
	if err := f.ctx.Redirect(url, statusCode); err != nil {
		logging.
			Error(f.Context()).
			Err(err).
			Msgf("Could not set set redirect %s %d", url, statusCode)
	}
}

func (f *fiberWebContext) StringBody() (string, error) {
	return string(f.ctx.Body()), nil
}

func (f *fiberWebContext) Path() string {
	return f.ctx.Path()
}

func (f *fiberWebContext) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	fileHeader, err := f.ctx.FormFile(key)

	if err != nil {
		return nil, nil, err
	}

	if file, err := fileHeader.Open(); err != nil {
		return nil, nil, err
	} else {
		return file, fileHeader, nil
	}
}

func (f *fiberWebContext) FormValue(key string) string {
	return f.ctx.FormValue(key)
}
