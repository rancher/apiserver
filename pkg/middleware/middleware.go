package middleware

import (
	"net/http"
)

// MiddlewareFunc is a function that wraps an http.Handler
type MiddlewareFunc func(http.Handler) http.Handler

// Chain is a slice of MiddlewareFunc
type Chain []MiddlewareFunc

// Handler applies the middleware chain to an http.Handler.
//
// The middleware are applied in the order they are defined in the Chain.
//
// That is, the first middleware in the Chain will be the outermost
// middleware that wraps the handler, and the last middleware will be the
// innermost middleware that directly wraps the handler.
func (m Chain) Handler(handler http.Handler) http.Handler {
	rtn := handler
	for i := len(m) - 1; i >= 0; i-- {
		w := m[i]
		rtn = w(rtn)
	}

	return rtn
}
