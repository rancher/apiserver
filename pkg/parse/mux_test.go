package parse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMuxURLParser(t *testing.T) {
	t.Run("parse all vars from mux", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/{prefix}/{type}/{namespace}/{name}/{link}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "nginx-pod", parsedURL.Name)
			assert.Equal(t, "default", parsedURL.Namespace)
			assert.Equal(t, "self", parsedURL.Link)
			assert.Equal(t, "v1", parsedURL.Prefix)
			assert.Equal(t, "GET", parsedURL.Method)
			assert.Empty(t, parsedURL.Action)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Get(server.URL + "/v1/pods/default/nginx-pod/self")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("parse with query parameters", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/{type}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "name=nginx", parsedURL.Query.Get("filter"))
			assert.Equal(t, "10", parsedURL.Query.Get("limit"))

			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Get(server.URL + "/pods?filter=name=nginx&limit=10")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("parse POST request", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("POST /{type}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "POST", parsedURL.Method)

			w.WriteHeader(http.StatusCreated)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Post(server.URL+"/pods", "application/json", nil)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("parse PUT request", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("PUT /{type}/{name}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "nginx-pod", parsedURL.Name)
			assert.Equal(t, "PUT", parsedURL.Method)

			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		req, err := http.NewRequest("PUT", server.URL+"/pods/nginx-pod", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("parse DELETE request", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("DELETE /{type}/{name}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "nginx-pod", parsedURL.Name)
			assert.Equal(t, "DELETE", parsedURL.Method)

			w.WriteHeader(http.StatusNoContent)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		req, err := http.NewRequest("DELETE", server.URL+"/pods/nginx-pod", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("parse with action", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/{type}/{name}", func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("action", "start")

			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "nginx-pod", parsedURL.Name)
			assert.Equal(t, "start", parsedURL.Action)

			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Get(server.URL + "/pods/nginx-pod")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("parse with no vars", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Empty(t, parsedURL.Type)
			assert.Empty(t, parsedURL.Name)
			assert.Empty(t, parsedURL.Namespace)
			assert.Empty(t, parsedURL.Link)
			assert.Empty(t, parsedURL.Prefix)
			assert.Empty(t, parsedURL.Action)
			assert.Equal(t, "GET", parsedURL.Method)

			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Get(server.URL + "/healthz")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("parse complex route with namespace", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/{prefix}/namespaces/{namespace}/{type}/{name}", func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := MuxURLParser(w, r, &types.APISchemas{})
			require.NoError(t, err)

			assert.Equal(t, "api", parsedURL.Prefix)
			assert.Equal(t, "default", parsedURL.Namespace)
			assert.Equal(t, "pods", parsedURL.Type)
			assert.Equal(t, "nginx-pod", parsedURL.Name)

			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(router)
		t.Cleanup(server.Close)

		resp, err := http.Get(server.URL + "/api/namespaces/default/pods/nginx-pod")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestVars(t *testing.T) {
	t.Run("create vars struct", func(t *testing.T) {
		vars := Vars{
			Type:      "deployment",
			Name:      "my-app",
			Namespace: "production",
			Link:      "logs",
			Prefix:    "/v1",
			Action:    "restart",
		}

		assert.Equal(t, "deployment", vars.Type)
		assert.Equal(t, "my-app", vars.Name)
		assert.Equal(t, "production", vars.Namespace)
		assert.Equal(t, "logs", vars.Link)
		assert.Equal(t, "/v1", vars.Prefix)
		assert.Equal(t, "restart", vars.Action)
	})

	t.Run("create empty vars", func(t *testing.T) {
		vars := Vars{}

		assert.Empty(t, vars.Type)
		assert.Empty(t, vars.Name)
		assert.Empty(t, vars.Namespace)
		assert.Empty(t, vars.Link)
		assert.Empty(t, vars.Prefix)
		assert.Empty(t, vars.Action)
	})
}
