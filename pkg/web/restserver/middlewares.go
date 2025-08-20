package restserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

const (
	authorizationHeader = "Authorization"
	userIDHeader        = "X-User-Id"
	tenantIDHeader      = "X-Tenant-Id"
)

type MiddlewareError struct {
	Err        error `json:"error"`
	StatusCode int   `json:"statusCode"`
}

func (e MiddlewareError) Error() string {
	return e.Err.Error()
}

func NewMiddlewareError(statusCode int, err error) *MiddlewareError {
	return &MiddlewareError{StatusCode: statusCode, Err: err}
}

type CustomMiddleware interface {
	Apply(ctx WebContext) *MiddlewareError
}

type CustomAuthenticationMiddleware interface {
	Apply(ctx WebContext) (*security.AuthenticationContext, error)
}

func authenticationContextFiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !strings.Contains(c.Request().URI().String(), string(AuthenticatedApi)) {
			return c.Next()
		}

		tenantID := string(c.Request().Header.Peek(tenantIDHeader))
		userID := string(c.Request().Header.Peek(userIDHeader))
		authCtx := security.NewAuthenticationContext(tenantID, userID)
		if authCtx.Valid() {
			newCtx := authCtx.SetInContext(c.UserContext())
			c.SetUserContext(newCtx)
			return c.Next()
		}

		c.Status(http.StatusUnauthorized)
		c.Request()
		return c.JSON(&Error{Error: "user not authenticated"})
	}
}

func customAuthenticationContextFiberMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		webCtx := &fiberWebContext{ctx: ctx}
		authCtx, err := customAuth.Apply(webCtx)
		if err != nil {
			ctx.Status(http.StatusUnauthorized)
			return ctx.JSON(err)
		}

		newCtx := authCtx.SetInContext(ctx.UserContext())
		ctx.SetUserContext(newCtx)
		return ctx.Next()
	}
}

func newRelicFiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		headers := make(http.Header)
		c.Context().Request.Header.VisitAll(func(key, value []byte) {
			headers.Set(string(key), string(value))
		})
		headers.Set("X-Request-URI", string(c.Request().RequestURI()))
		headers.Set("X-Protocol", c.Protocol())
		txn, ctx := monitoring.StartWebRequest(c.UserContext(), headers, c.Path(), c.Method())
		defer monitoring.EndTransaction(txn)

		c.SetUserContext(ctx)
		if err := c.Next(); err != nil {
			monitoring.NoticeError(txn, err)
			return err
		}

		monitoring.UpdateRequestSpan(txn, c.Method(), c.Route().Path)

		return nil
	}
	//return otelfiber.Middleware()
}

func accessControlFiberMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "OPTIONS, GET, POST, PUT, PATCH, DELETE",
		AllowHeaders: fmt.Sprintf("Origin, Content-Type, %s, %s, %s", authorizationHeader, userIDHeader, tenantIDHeader),
	})
}

func panicRecoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logging.Error(c.UserContext()).
					Err(fmt.Errorf("%v", r)).
					AddParam("path", c.Path()).
					AddParam("method", c.Method()).
					Msg("panic recovered")

				c.Status(fiber.StatusInternalServerError)
				err = c.JSON(Error{Error: "internal server error occurred"})
			}
		}()

		return c.Next()
	}
}
