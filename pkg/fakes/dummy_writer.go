package fakes

import "net/http"

// All other writers will attempt additional unnecessary logic
// Implements http.responseWriter and io.Writer
type DummyWriter struct {
	header map[string][]string
	buffer []byte
}

func NewDummyWriter() *DummyWriter {
	return &DummyWriter{map[string][]string{}, []byte{}}
}

func (d *DummyWriter) Header() http.Header {
	return d.header
}

func (d *DummyWriter) Buffer() []byte {
	return d.buffer
}

func (d *DummyWriter) Write(p []byte) (n int, err error) {
	d.buffer = append(d.buffer, p...)
	return 0, nil
}

func (d *DummyWriter) WriteHeader(int) {
}
