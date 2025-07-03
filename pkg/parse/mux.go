package parse

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
)

// MuxURLParser is a URL parser that uses path variables from `net/http` routing.
// This requires Go 1.22+ for path variable support in the standard library.
// It is named MuxURLParser for backward compatibility, but no longer uses gorilla/mux.
func MuxURLParser(rw http.ResponseWriter, req *http.Request, schemas *types.APISchemas) (ParsedURL, error) {
	url := ParsedURL{
		Type:      req.PathValue("type"),
		Name:      req.PathValue("name"),
		Namespace: req.PathValue("namespace"),
		Link:      req.PathValue("link"),
		Prefix:    req.PathValue("prefix"),
		Method:    req.Method,
		Action:    req.PathValue("action"),
		Query:     req.URL.Query(),
	}

	return url, nil
}
