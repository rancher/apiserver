package parse

import (
	"net/http"
	"strings"
)

func IsBrowser(req *http.Request, checkAccepts bool) bool {
	accepts := strings.ToLower(req.Header.Get("Accept"))
	userAgent := strings.ToLower(req.Header.Get("User-Agent"))

	if accepts == "" || !checkAccepts {
		accepts = "*/*"
	}

	// User agent has Mozilla and browser accepts */*
	return strings.Contains(userAgent, "mozilla") && strings.Contains(accepts, "*/*")
}

// MatchNotBrowserMiddleware returns a middleware that only allows non-browser requests
func MatchNotBrowserMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsBrowser(r, true) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Browser requests not allowed", http.StatusForbidden)
		})
	}
}

// MatchBrowserMiddleware returns a middleware that only allows browser requests
func MatchBrowserMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsBrowser(r, true) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Only browser requests allowed", http.StatusForbidden)
		})
	}
}
