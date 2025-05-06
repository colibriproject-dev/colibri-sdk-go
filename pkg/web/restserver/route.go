package restserver

import (
	"net/http"
)

// RoutePrefix is the type from default's routes
type RoutePrefix string

const (
	PublicApi        RoutePrefix = "/public/"
	PrivateApi       RoutePrefix = "/private/"
	AuthenticatedApi RoutePrefix = "/api/"
	NoPrefix         RoutePrefix = "/"
)

// Route represents an HTTP route with attributes for URI, method, prefix, handler function, and pre-execution middleware logic.
type Route struct {
	// URI specifies the path or endpoint for the route in the HTTP router.
	URI string
	// Method defines the HTTP method (e.g., GET, POST) associated with the route.
	Method string
	// Prefix specifies a common prefix for the routes.
	Prefix RoutePrefix
	// Function defines the handler function to be executed when the route is accessed.
	Function func(ctx WebContext)
	// BeforeEnter is a middleware function executed before entering the route handler; returns MiddlewareError for failures.
	BeforeEnter func(ctx WebContext) *MiddlewareError
}

type healtCheck struct {
	Status string `json:"status"`
}

func addHealthCheckRoute() {
	const route = "/health"
	srvRoutes = append(srvRoutes, Route{
		URI:    route,
		Method: http.MethodGet,
		Function: func(ctx WebContext) {
			ctx.JsonResponse(http.StatusOK, &healtCheck{"OK"})
		},
	})
}

func addDocumentationRoute() {
	const route = "/api-docs"
	srvRoutes = append(srvRoutes, Route{
		URI:    route,
		Method: http.MethodGet,
		Function: func(ctx WebContext) {
			ctx.ServeFile("./docs/swagger.json")
		},
	})
}
