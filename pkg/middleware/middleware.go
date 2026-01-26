package middleware

import (
	"net/http"
)

// MiddlewareFunc is a function that wraps an http.Handler
type MiddlewareFunc func(http.Handler) http.Handler

type Chain []MiddlewareFunc

func (m Chain) Handler(handler http.Handler) http.Handler {
	rtn := handler
	for i := len(m) - 1; i >= 0; i-- {
		w := m[i]
		rtn = w(rtn)
	}
	return rtn
}
