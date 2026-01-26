package parse

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetRouteVars(t *testing.T) {
	tests := []struct {
		name string
		vars RouteVars
	}{
		{
			name: "set single variable",
			vars: RouteVars{"id": "123"},
		},
		{
			name: "set multiple variables",
			vars: RouteVars{
				"id":   "456",
				"name": "test",
				"type": "resource",
			},
		},
		{
			name: "set empty variables",
			vars: RouteVars{},
		},
		{
			name: "set nil variables",
			vars: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Set route vars
			updatedReq := SetRouteVars(req, tt.vars)

			// Verify the request was updated
			if updatedReq == req {
				t.Error("SetRouteVars should return a new request with updated context")
			}

			// Verify vars can be retrieved
			retrievedVars := GetRouteVars(updatedReq)

			if tt.vars == nil {
				if len(retrievedVars) != 0 {
					t.Errorf("Expected empty vars for nil input, got %v", retrievedVars)
				}
				return
			}

			if len(retrievedVars) != len(tt.vars) {
				t.Errorf("Expected %d vars, got %d", len(tt.vars), len(retrievedVars))
			}

			for key, expectedVal := range tt.vars {
				if actualVal, ok := retrievedVars[key]; !ok {
					t.Errorf("Missing key %q in retrieved vars", key)
				} else if actualVal != expectedVal {
					t.Errorf("For key %q, expected %q, got %q", key, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestGetRouteVars(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
		expected RouteVars
	}{
		{
			name: "get existing variables",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return SetRouteVars(req, RouteVars{"id": "789", "name": "resource"})
			},
			expected: RouteVars{"id": "789", "name": "resource"},
		},
		{
			name: "get from request without variables",
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			expected: RouteVars{},
		},
		{
			name: "get empty variables",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				return SetRouteVars(req, RouteVars{})
			},
			expected: RouteVars{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			vars := GetRouteVars(req)

			if len(vars) != len(tt.expected) {
				t.Errorf("Expected %d vars, got %d", len(tt.expected), len(vars))
			}

			for key, expectedVal := range tt.expected {
				if actualVal, ok := vars[key]; !ok {
					t.Errorf("Missing key %q in retrieved vars", key)
				} else if actualVal != expectedVal {
					t.Errorf("For key %q, expected %q, got %q", key, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestRouteVars_IsolationBetweenRequests(t *testing.T) {
	// Create two requests with different variables
	req1 := httptest.NewRequest(http.MethodGet, "/resource/1", nil)
	req1 = SetRouteVars(req1, RouteVars{"id": "1"})

	req2 := httptest.NewRequest(http.MethodGet, "/resource/2", nil)
	req2 = SetRouteVars(req2, RouteVars{"id": "2"})

	// Verify isolation
	vars1 := GetRouteVars(req1)
	vars2 := GetRouteVars(req2)

	if vars1["id"] != "1" {
		t.Errorf("Expected req1 id to be '1', got %q", vars1["id"])
	}

	if vars2["id"] != "2" {
		t.Errorf("Expected req2 id to be '2', got %q", vars2["id"])
	}
}

func TestRouteVars_Immutability(t *testing.T) {
	// Set initial variables
	initialVars := RouteVars{"id": "initial"}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = SetRouteVars(req, initialVars)

	// Modify the original map
	initialVars["id"] = "modified"
	initialVars["new"] = "value"

	// Verify the request's variables are not affected
	retrievedVars := GetRouteVars(req)
	if retrievedVars["id"] != "initial" {
		t.Errorf("Expected id to remain 'initial', got %q", retrievedVars["id"])
	}

	if _, exists := retrievedVars["new"]; exists {
		t.Error("New key should not exist in request's variables")
	}
}

func TestMiddlewareFunc_CanWrapHandler(t *testing.T) {
	// Create a simple handler
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Create a middleware
	middlewareCalled := false
	middleware := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	// Wrap the handler with middleware
	wrappedHandler := middleware(handler)

	// Test the wrapped handler
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify both were called
	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMiddlewareFunc_ChainMultiple(t *testing.T) {
	// Track the order of execution
	var order []string

	// Create base handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Create first middleware
	middleware1 := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware1-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware1-after")
		})
	})

	// Create second middleware
	middleware2 := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware2-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware2-after")
		})
	})

	// Chain middlewares
	wrappedHandler := middleware1(middleware2(handler))

	// Execute
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify order of execution
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("Expected %d calls, got %d: %v", len(expectedOrder), len(order), order)
	}

	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("At position %d, expected %q, got %q", i, expected, order[i])
		}
	}
}
