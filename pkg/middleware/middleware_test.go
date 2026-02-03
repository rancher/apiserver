package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChainHandlerEmpty asserts that an empty chain returns the original handler.
func TestChainHandlerEmpty(t *testing.T) {
	assert := assert.New(t)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain{}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.True(called, "Original handler should be called")
	assert.Equal(http.StatusOK, w.Code)
}

// TestChainHandlerSingle asserts that a chain with a single middleware calls
// both middleware and handler.
func TestChainHandlerSingle(t *testing.T) {
	assert := assert.New(t)

	handlerCalled := false
	middlewareCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain{middleware1}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.True(middlewareCalled, "Middleware should be called")
	assert.True(handlerCalled, "Handler should be called")
	assert.Equal(http.StatusOK, w.Code)
}

// TestChainHandlerMultiple asserts that a chain with multiple middleware
// calls all middleware and the handler in the correct order.
func TestChainHandlerMultiple(t *testing.T) {
	assert := assert.New(t)

	var executionOrder []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-after")
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware3-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware3-after")
		})
	}

	chain := Chain{middleware1, middleware2, middleware3}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// The first middleware in the chain should be the outermost
	// Expected order: middleware1 -> middleware2 -> middleware3 -> handler
	// // -> middleware3 -> middleware2 -> middleware1
	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware3-before",
		"handler",
		"middleware3-after",
		"middleware2-after",
		"middleware1-after",
	}

	assert.Equal(expected, executionOrder, "Middleware should execute in the correct order")
	assert.Equal(http.StatusOK, w.Code)
}

// TestChainHandlerMiddlewareCanShortCircuit tests that middleware can prevent the handler from being called
func TestChainHandlerMiddlewareCanShortCircuit(t *testing.T) {
	assert := assert.New(t)

	handlerCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Middleware that short-circuits the request
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate authentication failure
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain{authMiddleware}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.False(handlerCalled, "Handler should not be called when middleware short-circuits")
	assert.Equal(http.StatusUnauthorized, w.Code)
}

// TestChainHandlerMiddlewareCanModifyRequest tests that middleware can modify the request
func TestChainHandlerMiddlewareCanModifyRequest(t *testing.T) {
	assert := assert.New(t)

	var receivedHeader string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom-Header")
		w.WriteHeader(http.StatusOK)
	})

	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Custom-Header", "custom-value")
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain{headerMiddleware}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("custom-value", receivedHeader, "Middleware should be able to modify the request")
	assert.Equal(http.StatusOK, w.Code)
}

// TestChainHandlerMiddlewareCanModifyResponse tests that middleware can modify the response
func TestChainHandlerMiddlewareCanModifyResponse(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("original response"))
	})

	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "added-by-middleware")
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain{headerMiddleware}
	wrappedHandler := chain.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("added-by-middleware", w.Header().Get("X-Custom-Header"), "Middleware should be able to modify response headers")
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("original response", w.Body.String())
}
