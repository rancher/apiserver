package apiserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

type Duck struct {
	Name string `json:"name"`
}

type DuckStore struct {
	empty.Store
}

func (s *DuckStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	return types.APIObjectList{
		Objects: []types.APIObject{
			{
				Type: schema.ID,
				ID:   "donald",
				Object: Duck{
					Name: "mallard",
				},
			},
			{
				Type: schema.ID,
				ID:   "howard",
				Object: Duck{
					Name: "teal",
				},
			},
		},
	}, nil
}

func (s *DuckStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	if id != "donald" {
		return types.APIObject{}, validation.NotFound
	}

	return types.APIObject{
		Type: schema.ID,
		ID:   "donald",
		Object: Duck{
			Name: "mallard",
		},
	}, nil
}

func (e *DuckStore) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	if id != "donald" {
		return types.APIObject{}, validation.NotFound
	}
	return types.APIObject{
		Type: schema.ID,
		ID:   "donald",
		Object: Duck{
			Name: "mallard",
		},
	}, nil
}

func (e *DuckStore) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	return types.APIObject{}, nil
}

func (e *DuckStore) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	return types.APIObject{}, nil
}

func TestDuckAPI_ListGetOnly(t *testing.T) {
	s := server.DefaultAPIServer()
	store := &DuckStore{}

	s.Schemas.MustImportAndCustomize(Duck{}, func(schema *types.APISchema) {
		schema.Store = store
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})

	router := http.NewServeMux()
	router.Handle("/{prefix}/{type}", s)
	router.Handle("/{prefix}/{type}/{name}", s)

	ts := httptest.NewServer(router)
	defer ts.Close()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		data, ok := body["data"]
		require.True(t, ok)

		items, ok := data.([]interface{})
		require.True(t, ok)
		require.Len(t, items, 2)
	}()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks/dynasty")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}()

	// Verify the others fail

	// Verify post fails
	func() {
		payload := []byte(`{"name":"Daffy"}`)
		bodyAsBytes, err := json.Marshal(payload)
		require.NoError(t, err)
		resp, err := http.Post(ts.URL+"/v1/ducks", "application/json", bytes.NewReader(bodyAsBytes))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	}()

	// Verify delete fails
	func() {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/ducks/donald", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	}()

	// Verify put and patch fail
	for _, requestMethod := range []string{http.MethodPut, http.MethodPatch} {
		func() {
			payload := []byte(`{"color":"teal"}`)
			req, err := http.NewRequest(requestMethod, ts.URL+"/v1/ducks/donald", bytes.NewReader(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()
			require.Equal(t, http.StatusForbidden, resp.StatusCode)
		}()
	}

	// Verify patch fails
	func() {
		payload := []byte(`{"color":"teal"}`)
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/v1/ducks/donald", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	}()
}

// Make sure that without resource/collection methods, this isn't accessible to the outside world.
func TestDuckAPI_EmptySchema_GetFails(t *testing.T) {
	s := server.DefaultAPIServer()
	store := &DuckStore{}

	s.Schemas.MustImportAndCustomize(Duck{}, func(schema *types.APISchema) {
		schema.Store = store
		schema.CollectionMethods = nil
		schema.ResourceMethods = nil
	})

	router := http.NewServeMux()
	router.Handle("/{prefix}/{type}", s)
	router.Handle("/{prefix}/{type}/{name}", s)

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/ducks")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/v1/ducks/donald")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestDuckAPI_AllMethodsSupported(t *testing.T) {
	s := server.DefaultAPIServer()
	store := &DuckStore{}

	s.Schemas.MustImportAndCustomize(Duck{}, func(schema *types.APISchema) {
		schema.Store = store
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete}
	})

	router := http.NewServeMux()
	router.Handle("/{prefix}/{type}", s)
	router.Handle("/{prefix}/{type}/{name}", s)

	ts := httptest.NewServer(router)
	defer ts.Close()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		data, ok := body["data"]
		require.True(t, ok)

		items, ok := data.([]interface{})
		require.True(t, ok)
		require.Len(t, items, 2)
	}()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}()

	func() {
		resp, err := http.Get(ts.URL + "/v1/ducks/dynasty")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}()

	// Verify post works
	func() {
		payload := []byte(`{"name":"Daffy"}`)
		//bodyAsBytes, err := json.Marshal(payload)
		//require.NoError(t, err)
		resp, err := http.Post(ts.URL+"/v1/ducks", "application/json", bytes.NewReader(payload))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}()

	// Verify delete works
	func() {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/ducks/donald", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}()

	// Verify put works
	func() {
		payload := []byte(`{"color":"teal"}`)
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/v1/ducks/donald", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}()
}
