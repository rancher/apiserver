package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContentTypeAutoDetectHTML tests that HTML content type is auto-detected
func TestContentTypeAutoDetectHTML(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Hello</body></html>"))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Contains(w.Header().Get("Content-Type"), "text/html",
		"Content-Type should be auto-detected as HTML")
	assert.Equal("<!DOCTYPE html><html><body>Hello</body></html>", w.Body.String())
}

// TestContentTypeAutoDetectJSON tests that JSON content type is auto-detected
func TestContentTypeAutoDetectJSON(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message": "hello"}`))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/api/data", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// http.DetectContentType detects JSON as text/plain
	assert.Contains(w.Header().Get("Content-Type"), "text/plain",
		"Content-Type should be set")
	assert.Equal(`{"message": "hello"}`, w.Body.String())
}

// TestContentTypeAutoDetectPlainText tests plain text detection
func TestContentTypeAutoDetectPlainText(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain text content"))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Contains(w.Header().Get("Content-Type"), "text/plain",
		"Content-Type should be auto-detected as plain text")
}

// TestContentTypePresetHeaderNotOverwritten tests that existing Content-Type is not overwritten
func TestContentTypePresetHeaderNotOverwritten(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("plain text content"))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/api/data", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal("application/json; charset=utf-8", w.Header().Get("Content-Type"),
		"Pre-set Content-Type should not be overwritten")
	assert.Equal("plain text content", w.Body.String())
}

// TestContentTypeCaseInsensitiveHeader tests case-insensitive header check
func TestContentTypeCaseInsensitiveHeader(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name       string
		headerName string
	}{
		{"lowercase", "content-type"},
		{"uppercase", "CONTENT-TYPE"},
		{"mixedcase", "Content-Type"},
		{"weirdcase", "CoNtEnT-TyPe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(tt.headerName, "application/custom")
				_, _ = w.Write([]byte("test"))
			})

			wrappedHandler := ContentType(handler)

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal("application/custom", w.Header().Get("Content-Type"),
				"Content-Type with case %s should not be overwritten", tt.headerName)
		})
	}
}

// TestContentTypeMultipleWrites tests behavior with multiple Write calls
func TestContentTypeMultipleWrites(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("First "))
		_, _ = w.Write([]byte("Second "))
		_, _ = w.Write([]byte("Third"))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.NotEmpty(w.Header().Get("Content-Type"),
		"Content-Type should be set on first Write")
	assert.Equal("First Second Third", w.Body.String())
}

// TestContentTypeWriteHeaderBeforeWrite tests WriteHeader called before Write
func TestContentTypeWriteHeaderBeforeWrite(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("content"))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(http.StatusCreated, w.Code)
	assert.NotEmpty(w.Header().Get("Content-Type"),
		"Content-Type should still be set when WriteHeader is called first")
	assert.Equal("content", w.Body.String())
}

// TestContentTypeNoWrite tests when handler doesn't write anything
func TestContentTypeNoWrite(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No Write call
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("DELETE", "/resource", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(http.StatusNoContent, w.Code)
}

// TestContentTypeEmptyWrite tests writing empty content
func TestContentTypeEmptyWrite(t *testing.T) {
	assert := assert.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	})

	wrappedHandler := ContentType(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Empty content will be detected as text/plain; charset=utf-8
	assert.NotEmpty(w.Header().Get("Content-Type"),
		"Content-Type should be set even for empty writes")
}

// TestContentTypeWriterHijack tests the Hijack method implementation
func TestContentTypeWriterHijack(t *testing.T) {
	assert := assert.New(t)

	// Test with a ResponseWriter that doesn't implement Hijacker
	w := httptest.NewRecorder()
	writer := ContentTypeWriter{ResponseWriter: w}

	conn, rw, err := writer.Hijack()

	assert.Nil(conn, "Connection should be nil when hijacking is not supported")
	assert.Nil(rw, "ReadWriter should be nil when hijacking is not supported")
	assert.ErrorContains(err, "does not implement http.Hijacker")
}

// TestContentTypeWriterHijackSuccess tests successful hijacking
func TestContentTypeWriterHijackSuccess(t *testing.T) {
	assert := assert.New(t)

	mock := &fakeHijackableWriter{ResponseRecorder: httptest.NewRecorder()}
	writer := ContentTypeWriter{ResponseWriter: mock}

	conn, rw, err := writer.Hijack()

	assert.True(mock.hijackCalled, "Hijack should be called on underlying writer")
	assert.NoError(err, "Should not return error when underlying writer supports Hijacker")

	assert.Nil(conn)
	assert.Nil(rw)
}

// TestContentTypeWithDifferentContentTypes tests various content types
func TestContentTypeWithDifferentContentTypes(t *testing.T) {
	tests := []struct {
		name             string
		content          []byte
		expectedContains string
	}{
		{
			name:             "HTML",
			content:          []byte("<!DOCTYPE html><html></html>"),
			expectedContains: "text/html",
		},
		{
			name:             "XML",
			content:          []byte("<?xml version=\"1.0\"?><root></root>"),
			expectedContains: "text/xml",
		},
		{
			name:             "Plain text",
			content:          []byte("simple text"),
			expectedContains: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(tt.content)
			})

			wrappedHandler := ContentType(handler)

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Contains(w.Header().Get("Content-Type"), tt.expectedContains,
				"Content-Type should contain %s", tt.expectedContains)
			assert.Equal(tt.content, w.Body.Bytes())
		})
	}
}

// fakeHijackableWriter is a mock that implements http.Hijacker
type fakeHijackableWriter struct {
	*httptest.ResponseRecorder
	hijackCalled bool
}

func (m *fakeHijackableWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijackCalled = true
	return nil, nil, nil
}
