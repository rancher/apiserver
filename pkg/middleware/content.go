package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"reflect"
)

// ContentTypeWriter is a custom ResponseWriter that sets the Content-Type
// header if it is not already set.
type ContentTypeWriter struct {
	http.ResponseWriter
}

// Write sets the Content-Type header if not already set, then writes the response.
func (c ContentTypeWriter) Write(b []byte) (int, error) {
	found := c.Header().Get("Content-Type") != ""
	if !found {
		c.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return c.ResponseWriter.Write(b)
}

// ContentType is a middleware that sets the Content-Type header if it is not already set.
func ContentType(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := ContentTypeWriter{ResponseWriter: w}
		handler.ServeHTTP(writer, r)
	})
}

func (c ContentTypeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("upstream ResponseWriter of type %v does not implement http.Hijacker", reflect.TypeOf(c.ResponseWriter))
}
