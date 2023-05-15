package middleware

import (
	"compress/gzip"
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/stretchr/testify/assert"
)

func NewRequest(accept string) *http.Request {
	return &http.Request{
		Header: map[string][]string{"Accept-Encoding": {accept}},
	}
}

// TestWriteHeader asserts content-length header is deleted and content-encoding header is set to gzip
func TestWriteHeader(t *testing.T) {
	assert := assert.New(t)

	w := fakes.NewDummyWriter()
	gz := &gzipResponseWriter{gzip.NewWriter(w), w}

	gz.Header().Set("Content-Length", "80")
	gz.WriteHeader(400)
	// Content-Length should have been deleted in WriterHeader, resulting in empty string
	assert.Equal("", gz.Header().Get("Content-Length"))
	assert.Equal(1, len(w.Header()["Content-Encoding"]))
	assert.Equal("gzip", gz.Header().Get("Content-Encoding"))
}

// TestSetContentWithoutWrite asserts content-encoding is NOT "gzip" if accept-encoding header does not contain gzip
func TestSetContentWithoutWrite(t *testing.T) {
	assert := assert.New(t)

	// Test content encoding header when write is not used
	handlerFunc := Gzip(&fakes.DummyHandler{})

	// Test when accept-encoding only contains gzip
	rw := fakes.NewDummyWriter()
	req := NewRequest("gzip")
	handlerFunc.ServeHTTP(rw, req)
	// Content encoding should be empty since write has not been used
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding contains multiple options, including gzip
	rw = fakes.NewDummyWriter()
	req = NewRequest("json, xml, gzip")
	handlerFunc.ServeHTTP(rw, req)
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding is empty
	req = NewRequest("")
	rw = fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding is is not empty but does not include gzip
	req = NewRequest("json, xml")
	rw = fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))
}

// TestSetContentWithWrite asserts content-encoding is "gzip" if accept-encoding header contains gzip
func TestSetContentWithWrite(t *testing.T) {
	assert := assert.New(t)

	// Test content encoding header when write is used
	handlerFunc := Gzip(&fakes.DummyHandlerWithWrite{})

	// Test when accept-encoding only contains gzip
	req := NewRequest("gzip")
	rw := fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	// Content encoding should be gzip since write has been used
	assert.Equal(1, len(rw.Header()["Content-Encoding"]))
	assert.Equal("gzip", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding contains multiple options, including gzip
	req = NewRequest("json, xml, gzip")
	rw = fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	// Content encoding should be gzip since write has been used
	assert.Equal(1, len(rw.Header()["Content-Encoding"]))
	assert.Equal("gzip", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding is empty
	req = NewRequest("")
	rw = fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	// Content encoding should be empty since gzip is not an accepted encoding
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))

	// Test when accept-encoding is is not empty but does not include gzip
	req = NewRequest("json, xml")
	rw = fakes.NewDummyWriter()
	handlerFunc.ServeHTTP(rw, req)
	// Content encoding should be empty since gzip is not an accepted encoding
	assert.Equal(0, len(rw.Header()["Content-Encoding"]))
	assert.Equal("", rw.Header().Get("Content-Encoding"))
}

// TestMultipleWrites ensures that Write can be used multiple times
func TestMultipleWrites(t *testing.T) {
	assert := assert.New(t)

	// Handler function that contains one writing handler
	handlerFuncOneWrite := Gzip(&fakes.DummyHandlerWithWrite{})

	// Handler function that contains a chain of two writing handlers
	handlerFuncTwoWrites := Gzip(fakes.NewDummyHandlerWithWrite(&fakes.DummyHandlerWithWrite{}))

	req := NewRequest("gzip")
	rw := fakes.NewDummyWriter()
	handlerFuncOneWrite.ServeHTTP(rw, req)
	oneWriteResult := rw.Buffer()

	req = NewRequest("gzip")
	rw = fakes.NewDummyWriter()
	handlerFuncTwoWrites.ServeHTTP(rw, req)
	multiWriteResult := rw.Buffer()

	// Content encoding should be gzip since write has been used (twice)
	assert.Equal(1, len(rw.Header()["Content-Encoding"]))
	assert.Equal("gzip", rw.Header().Get("Content-Encoding"))
	assert.NotEqual(multiWriteResult, oneWriteResult)
}
