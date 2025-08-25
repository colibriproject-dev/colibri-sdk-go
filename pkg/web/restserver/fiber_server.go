package restserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type fiberWebServer struct {
	srv *fiber.App
}

func createFiberServer() Server {
	return &fiberWebServer{}
}

func (f *fiberWebServer) initialize() {
	f.srv = fiber.New(fiber.Config{
		ServerHeader:          "colibri-sdk-go",
		AppName:               config.APP_NAME,
		DisableStartupMessage: true,
	})
}

func (f *fiberWebServer) shutdown() error {
	return f.srv.ShutdownWithTimeout(10 * time.Second)
}

func (f *fiberWebServer) injectMiddlewares() {
	if monitoring.UseOTELMonitoring() {
		f.srv.Use(newOpenTelemetryFiberMiddleware())
	}
	f.srv.Use(accessControlFiberMiddleware())
	f.srv.Use(panicRecoverMiddleware())
	if customAuth != nil {
		f.srv.Use(customAuthenticationContextFiberMiddleware())
	} else {
		f.srv.Use(authenticationContextFiberMiddleware())
	}
}

func (f *fiberWebServer) injectCustomMiddlewares() {
	for _, middleware := range customMiddlewares {
		f.registerCustomMiddleware(middleware)
	}
}

func (f *fiberWebServer) convertUriToFiberUri(uri string, replacer *strings.Replacer) string {
	paths := strings.Split(uri, "/")

	for idx, path := range paths {
		if f.pathIsPathParam(path) {
			paths[idx] = fmt.Sprintf(":%s", replacer.Replace(path))
		}
	}

	return strings.Join(paths, "/")
}

func (f *fiberWebServer) pathIsPathParam(path string) bool {
	return strings.Contains(path, "{")
}

func (f *fiberWebServer) injectRoutes() {
	f.addMetricsRoute()
	f.addSwaggerUI()

	replacer := strings.NewReplacer(
		"{", "",
		"}", "",
	)

	for _, route := range srvRoutes {
		routeUri := string(route.Prefix) + f.convertUriToFiberUri(route.URI, replacer)
		fn := route.Function
		beforeEnter := route.BeforeEnter

		f.srv.Add(route.Method, routeUri, func(fctx *fiber.Ctx) error {
			fctx.Set(parameterizedURLHeaderKey, routeUri)
			webContext := newFiberWebContext(fctx)
			if beforeEnter != nil {
				if err := beforeEnter(webContext); err != nil {
					fctx.Status(err.StatusCode)
					return fctx.JSON(Error{err.Err.Error()})
				}
			}

			fn(webContext)
			return nil
		}).Name(routeUri)

		logging.
			Info(context.Background()).
			Msgf("Registered route [%7s] %s", route.Method, string(route.Prefix)+route.URI)
	}
}

func (f *fiberWebServer) listenAndServe() error {
	return f.srv.Listen(fmt.Sprintf(":%d", config.PORT))
}

func (f *fiberWebServer) addMetricsRoute() {
	const route = "/metrics"

	p := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	f.srv.Get(route, func(c *fiber.Ctx) error {
		p(c.Context())
		return nil
	})
}

func (f *fiberWebServer) addSwaggerUI() {
	if config.IsDevelopmentEnvironment() {
		f.srv.Get("/swagger/*", swagger.New(swagger.Config{URL: "/api-docs"}))
	}
}

func (f *fiberWebServer) registerCustomMiddleware(m CustomMiddleware) {
	fn := func(ctx *fiber.Ctx) error {
		webCtx := &fiberWebContext{ctx: ctx}
		if err := m.Apply(webCtx); err != nil {
			ctx.Status(err.StatusCode)
			return ctx.JSON(Error{err.Err.Error()})
		}

		return ctx.Next()
	}

	f.srv.Use(fn)
}
