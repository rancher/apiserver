// nolint: errcheck
package apiserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

type Duck struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DuckStore struct {
	empty.Store
	ducks []Duck
}

var basicDucks = []Duck{
	{
		ID:   "howard",
		Name: "teal",
	},
	{
		ID:   "donald",
		Name: "mallard",
	},
}

func NewDuckStore(initialDucks []Duck) *DuckStore {
	if len(initialDucks) == 0 {
		initialDucks = basicDucks
	}
	return &DuckStore{ducks: initialDucks}
}

func (s *DuckStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	var returnedObjects []types.APIObject
	for _, duck := range s.ducks {
		returnedObjects = append(returnedObjects, types.APIObject{
			Type: schema.ID,
			ID:   duck.ID,
			Object: Duck{
				Name: duck.Name,
			},
		})
	}
	return types.APIObjectList{Objects: returnedObjects}, nil
}

func (s *DuckStore) getDuckIndex(id string) int {
	return slices.IndexFunc(s.ducks, func(d Duck) bool {
		return d.ID == id
	})
}

func (s *DuckStore) ByID(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getDuckIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	duck := s.ducks[index]
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Duck{
			Name: duck.Name,
		},
	}, nil
}

func (s *DuckStore) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getDuckIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	duckToDelete := s.ducks[index]
	s.ducks = slices.Delete(s.ducks, index, index+1)
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Duck{
			Name: duckToDelete.Name,
		},
	}, nil
}

func (s *DuckStore) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	s.ducks = append(s.ducks, Duck{ID: data.ID, Name: data.Object.(map[string]any)["name"].(string)})
	return data, nil
}

func (s *DuckStore) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	index := s.getDuckIndex(id)
	if index == -1 {
		return data, validation.NotFound
	}
	newName, ok := data.Object.(map[string]any)["name"].(string)
	if !ok {
		return data, validation.NotFound
	}
	s.ducks[index].Name = newName
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Duck{
			Name: newName,
		},
	}, nil
}

func TestDuckAPI_ListGetOnly(t *testing.T) {
	s := server.DefaultAPIServer()
	store := NewDuckStore(nil)

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

	t.Run("can list ducks", func(t *testing.T) {
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
	})

	t.Run("can get ducks by ID", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("can't get unknown duck", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/dynasty")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Verify the others fail

	t.Run("can't create ducks", func(t *testing.T) {
		payload := []byte(`{"name":"Daffy"}`)
		bodyAsBytes, err := json.Marshal(payload)
		require.NoError(t, err)
		resp, err := http.Post(ts.URL+"/v1/ducks", "application/json", bytes.NewReader(bodyAsBytes))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("can't delete ducks", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/ducks/donald", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	for _, requestMethod := range []string{http.MethodPut, http.MethodPatch} {
		requestMethod := requestMethod
		t.Run(fmt.Sprintf("can't modify existing ducks with %s", requestMethod), func(t *testing.T) {
			payload := []byte(`{"color":"teal"}`)
			req, err := http.NewRequest(requestMethod, ts.URL+"/v1/ducks/donald", bytes.NewReader(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()
			require.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// Make sure that without resource/collection methods, this isn't accessible to the outside world.
func TestDuckAPI_EmptySchema_GetFails(t *testing.T) {
	s := server.DefaultAPIServer()
	store := NewDuckStore(nil)

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

	t.Run("can list ducks", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("can get ducks by ID", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestDuckAPI_AllMethodsSupported(t *testing.T) {
	s := server.DefaultAPIServer()
	store := NewDuckStore(nil)

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

	t.Run("can list ducks", func(t *testing.T) {
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
	})

	t.Run("can get ducks by ID", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("can't get a duck we don't have", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/dynasty")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("can create ducks", func(t *testing.T) {
		payload := []byte(`{"id":"daffy", "name":"orange"}`)
		resp, err := http.Post(ts.URL+"/v1/ducks", "application/json", bytes.NewReader(payload))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("can delete an existing duck", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/ducks/donald", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("can't delete a non-existing duck", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/ducks/wallace", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("can't get a deleted duck", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/ducks/donald")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("can modify ducks", func(t *testing.T) {
		payload := []byte(`{"name":"teal"}`)
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/v1/ducks/howard", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
