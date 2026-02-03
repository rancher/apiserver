package middleware

import (
	"net/http"
	"strings"
)

// CacheMiddleware sets Cache-Control headers for specified file suffixes.
func CacheMiddleware(suffixes ...string) MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return Cache(handler, suffixes...)
	}
}

// Cache wraps an http.Handler to set Cache-Control headers for specified
// file suffixes.
//
// e.g., Cache(handler, "js", "css") will set
// "Cache-Control: max-age=31536000, public" for requests ending with .js or .css
func Cache(handler http.Handler, suffixes ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := strings.LastIndex(r.URL.Path, ".")
		if i >= 0 {
			for _, suffix := range suffixes {
				if suffix == r.URL.Path[i+1:] {
					w.Header().Set("Cache-Control", "max-age=31536000, public")
				}
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// NoCache sets no-cache headers on responses.
func NoCache(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		handler.ServeHTTP(w, r)
	})
}

// FrameOptions middleware sets X-Frame-Options header on responses.
func FrameOptions(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		handler.ServeHTTP(w, r)
	})
}

// ContentTypeOptions middleware sets X-Content-Type-Options header on responses.
func ContentTypeOptions(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		handler.ServeHTTP(w, r)
	})
}
