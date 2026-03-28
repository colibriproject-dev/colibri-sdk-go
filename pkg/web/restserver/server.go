package restserver

import (
	"context"
	"errors"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

var (
	srvRoutes         []Route
	customMiddlewares []CustomMiddleware
	srv               Server
	customAuth        CustomAuthenticationMiddleware

	errUserUnauthenticated = errors.New("user not authenticated")
)

// Server defines the contract for the HTTP server implementation.
type Server interface {
	// initialize prepares the server for running.
	initialize()
	// shutdown gracefully closes the server.
	shutdown() error
	// injectMiddlewares adds standard middlewares to the server.
	injectMiddlewares()
	// injectCustomMiddlewares adds user-defined middlewares to the server.
	injectCustomMiddlewares()
	// injectRoutes registers all routes in the server.
	injectRoutes()
	// listenAndServe starts the server and listens for requests.
	listenAndServe() error
}

// AddRoutes adds a list of routes to the web rest server.
func AddRoutes(routes []Route) {
	srvRoutes = append(srvRoutes, routes...)
}

// CustomAuthMiddleware adds a custom authentication middleware to the web server.
func CustomAuthMiddleware(fn CustomAuthenticationMiddleware) {
	customAuth = fn
}

// Use adds a custom middleware to the web server.
func Use(m CustomMiddleware) {
	customMiddlewares = append(customMiddlewares, m)
}

// ListenAndServe initializes, configures, and starts the web rest server.
func ListenAndServe() {
	addHealthCheckRoute()
	addDocumentationRoute()

	srv = createFiberServer()
	srv.initialize()
	srv.injectMiddlewares()
	srv.injectCustomMiddlewares()
	srv.injectRoutes()

	observer.Attach(restObserver{})
	logging.Info(context.Background()).Msgf("Service '%s' running in %d port", "WEB-REST", config.PORT)
	if err := srv.listenAndServe(); err != nil {
		logging.
			Fatal(context.Background()).
			Err(err).
			Msg("Error on trying to initialize rest server")
	}
}
