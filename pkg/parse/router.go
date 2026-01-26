package parse

import (
	"context"
	"maps"
	"net/http"
)

type ctxKey struct{}

// RouteVars stores the route variables for a request.
type RouteVars map[string]string

// SetRouteVars sets route variables in the request context.
func SetRouteVars(r *http.Request, vars RouteVars) *http.Request {
	copiedVars := maps.Clone(vars)
	return r.WithContext(context.WithValue(r.Context(), ctxKey{}, copiedVars))
}

// GetRouteVars retrieves route variables from the request context.
func GetRouteVars(r *http.Request) RouteVars {
	if vars, ok := r.Context().Value(ctxKey{}).(RouteVars); ok {
		return vars
	}

	return RouteVars{}
}

// MiddlewareFunc is a function that wraps an http.Handler
type MiddlewareFunc func(http.Handler) http.Handler
