package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCacheMiddleware tests the CacheMiddleware function.
//
// It verifies that the middleware sets the correct Cache-Control header for
// requests with specified file suffixes.
func TestCacheMiddleware(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CacheMiddleware("js", "css")
	wrappedHandler := middleware(handler)

	// Test with .js file
	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("max-age=31536000, public", w.Header().Get("Cache-Control"))
	assert.Equal(http.StatusOK, w.Code)
}

// TestCacheMatchingSuffix tests that Cache sets headers for matching file suffixes
func TestCacheMatchingSuffix(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Cache(handler, "js", "css", "png")

	tests := []struct {
		name string
		path string
	}{
		{"JavaScript file", "/static/app.js"},
		{"CSS file", "/styles/main.css"},
		{"PNG image", "/images/logo.png"},
		{"Nested path with JS", "/vendor/lib/script.js"},
		{"Path with multiple dots", "/file.min.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("max-age=31536000, public", w.Header().Get("Cache-Control"),
				"Cache-Control should be set for %s", tt.path)
			assert.Equal(http.StatusOK, w.Code)
		})
	}
}

// TestCacheNonMatchingSuffix tests that Cache does not set headers for non-matching suffixes
func TestCacheNonMatchingSuffix(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Cache(handler, "js", "css")

	tests := []struct {
		name string
		path string
	}{
		{"HTML file", "/index.html"},
		{"JSON file", "/api/data.json"},
		{"No extension", "/path/without/extension"},
		{"Different extension", "/document.pdf"},
		{"Root path", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("", w.Header().Get("Cache-Control"),
				"Cache-Control should not be set for %s", tt.path)
			assert.Equal(http.StatusOK, w.Code)
		})
	}
}

// TestCacheEmptySuffixes tests Cache with no suffixes provided
func TestCacheEmptySuffixes(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Cache(handler)

	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("", w.Header().Get("Cache-Control"),
		"Cache-Control should not be set when no suffixes are provided")
	assert.Equal(http.StatusOK, w.Code)
}

// TestCacheCaseSensitive tests that suffix matching is case-sensitive
func TestCacheCaseSensitive(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Cache(handler, "js")

	// Test lowercase (should match)
	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	assert.Equal("max-age=31536000, public", w.Header().Get("Cache-Control"))

	// Test uppercase (should not match)
	req = httptest.NewRequest("GET", "/app.JS", nil)
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	assert.Equal("", w.Header().Get("Cache-Control"),
		"Suffix matching should be case-sensitive")
}

// TestCacheHandlerStillCalled tests that the wrapped handler is always called
func TestCacheHandlerStillCalled(t *testing.T) {
	assert := assert.New(t)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Cache(handler, "js")

	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.True(handlerCalled, "Wrapped handler should be called")
}

// TestNoCache tests the NoCache middleware
func TestNoCache(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	wrappedHandler := NoCache(handler)

	tests := []struct {
		name string
		path string
	}{
		{"Root path", "/"},
		{"API endpoint", "/api/data"},
		{"With extension", "/file.html"},
		{"Nested path", "/a/b/c/d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"),
				"NoCache should set Cache-Control header for %s", tt.path)
			assert.Equal(http.StatusOK, w.Code)
			assert.Equal("response", w.Body.String())
		})
	}
}

// TestFrameOptions tests the FrameOptions middleware
func TestFrameOptions(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	wrappedHandler := FrameOptions(handler)

	tests := []struct {
		name string
		path string
	}{
		{"Root path", "/"},
		{"HTML page", "/index.html"},
		{"Nested path", "/admin/dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("SAMEORIGIN", w.Header().Get("X-Frame-Options"),
				"FrameOptions should set X-Frame-Options header for %s", tt.path)
			assert.Equal(http.StatusOK, w.Code)
			assert.Equal("response", w.Body.String())
		})
	}
}

// TestContentTypeOptions tests the ContentTypeOptions middleware
func TestContentTypeOptions(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	wrappedHandler := ContentTypeOptions(handler)

	tests := []struct {
		name string
		path string
	}{
		{"Root path", "/"},
		{"JavaScript file", "/app.js"},
		{"API endpoint", "/api/users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("nosniff", w.Header().Get("X-Content-Type-Options"),
				"ContentTypeOptions should set X-Content-Type-Options header for %s", tt.path)
			assert.Equal(http.StatusOK, w.Code)
			assert.Equal("response", w.Body.String())
		})
	}
}

// TestMiddlewareCombination tests combining multiple middleware functions
func TestMiddlewareCombination(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	// Combine multiple middleware
	wrappedHandler := NoCache(FrameOptions(ContentTypeOptions(handler)))

	req := httptest.NewRequest("GET", "/secure/page", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Equal("SAMEORIGIN", w.Header().Get("X-Frame-Options"))
	assert.Equal("nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("response", w.Body.String())
}
