package parse

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
)

// Vars represents route variables that can be set in the request context.
type Vars struct {
	Type      string
	Name      string
	Namespace string
	Link      string
	Prefix    string
	Action    string
}

// SetVarsMiddleware creates a middleware that sets route variables in the request context
func SetVarsMiddleware(v Vars) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := GetRouteVars(r)
			if vars == nil {
				vars = RouteVars{}
			}

			if v.Type != "" {
				vars["type"] = v.Type
			}
			if v.Name != "" {
				vars["name"] = v.Name
			}
			if v.Link != "" {
				vars["link"] = v.Link
			}
			if v.Prefix != "" {
				vars["prefix"] = v.Prefix
			}
			if v.Action != "" {
				vars["action"] = v.Action
			}
			if v.Namespace != "" {
				vars["namespace"] = v.Namespace
			}

			r = SetRouteVars(r, vars)
			next.ServeHTTP(w, r)
		})
	}
}

// MuxURLParser is a URLParser implementation for the standard library's ServeMux router.
//
// It extracts route variables from the request and constructs a ParsedURL
// accordingly.
func MuxURLParser(rw http.ResponseWriter, req *http.Request, schemas *types.APISchemas) (ParsedURL, error) {
	// Get path parameters from the new router (Go 1.22+ pattern matching)
	vars := RouteVars{
		"type":      req.PathValue("type"),
		"name":      req.PathValue("name"),
		"namespace": req.PathValue("namespace"),
		"link":      req.PathValue("link"),
		"prefix":    req.PathValue("prefix"),
		"action":    req.PathValue("action"),
	}

	// Also check context vars set by SetVarsMiddleware
	contextVars := GetRouteVars(req)
	for k, v := range contextVars {
		if v != "" {
			vars[k] = v
		}
	}

	url := ParsedURL{
		Type:      vars["type"],
		Name:      vars["name"],
		Namespace: vars["namespace"],
		Link:      vars["link"],
		Prefix:    vars["prefix"],
		Method:    req.Method,
		Action:    vars["action"],
		Query:     req.URL.Query(),
	}

	return url, nil
}
