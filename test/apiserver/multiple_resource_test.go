// nolint: errcheck
package apiserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/apiroot"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type PetStore struct {
	empty.Store
	mu       sync.Mutex
	watchers []chan types.APIEvent
}

type Dog struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type DogStore struct {
	PetStore
	dogs     []Dog
}

type Cat struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type CatStore struct {
	PetStore
	cats     []Cat
}

// Common pet store operations

func (s *PetStore) broadcast(event types.APIEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.watchers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *PetStore) Watch(apiOp *types.APIRequest, _ *types.APISchema, _ types.WatchRequest) (chan types.APIEvent, error) {
	ch := make(chan types.APIEvent, 100)
	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()
	go func() {
		<-apiOp.Request.Context().Done()
		s.mu.Lock()
		for i, w := range s.watchers {
			if w == ch {
				s.watchers = slices.Delete(s.watchers, i, i+1)
				break
			}
		}
		s.mu.Unlock()
		close(ch)
	}()
	return ch, nil
}

var basicDogs = []Dog{
	{ID: "pluto", Name: "disney"},
	{ID: "krypto", Name: "dc"},
}

func NewDogStore(initialDogs []Dog) *DogStore {
	if len(initialDogs) == 0 {
		initialDogs = basicDogs
	}
	return &DogStore{dogs: initialDogs}
}

func (s *DogStore) getDogIndex(id string) int {
	return slices.IndexFunc(s.dogs, func(d Dog) bool { return d.ID == id })
}

func (s *DogStore) Create(_ *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	newDog := Dog{ID: data.ID, Name: data.Object.(map[string]any)["name"].(string)}
	s.mu.Lock()
	s.dogs = append(s.dogs, newDog)
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: newDog.ID, Object: Dog{Name: newDog.Name}}
	s.broadcast(types.APIEvent{Name: types.CreateAPIEvent, Object: result})
	return result, nil
}

func (s *DogStore) Update(_ *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	s.mu.Lock()
	i := s.getDogIndex(id)
	if i == -1 {
		s.mu.Unlock()
		return types.APIObject{}, validation.NotFound
	}
	newName := data.Object.(map[string]any)["name"].(string)
	s.dogs[i].Name = newName
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: id, Object: Dog{Name: newName}}
	s.broadcast(types.APIEvent{Name: types.ChangeAPIEvent, Object: result})
	return result, nil
}

func (s *DogStore) Delete(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	s.mu.Lock()
	i := s.getDogIndex(id)
	if i == -1 {
		s.mu.Unlock()
		return types.APIObject{}, validation.NotFound
	}
	dog := s.dogs[i]
	s.dogs = slices.Delete(s.dogs, i, i+1)
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: id, Object: Dog{Name: dog.Name}}
	s.broadcast(types.APIEvent{Name: types.RemoveAPIEvent, Object: result})
	return result, nil
}

func (s *DogStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	objects := make([]types.APIObject, len(s.dogs))
	for i, dog := range s.dogs {
		objects[i] = types.APIObject{Type: schema.ID, ID: dog.ID, Object: Dog{Name: dog.Name}}
	}
	return types.APIObjectList{Objects: objects}, nil
}

func (s *DogStore) ByID(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	i := s.getDogIndex(id)
	if i == -1 {
		return types.APIObject{}, validation.NotFound
	}
	return types.APIObject{Type: schema.ID, ID: id, Object: Dog{Name: s.dogs[i].Name}}, nil
}

var basicCats = []Cat{
	{ID: "felix", Name: "dell"},
	{ID: "fritz", Name: "zap"},
	{ID: "boris", Name: "home"},
}

func NewCatStore(initialCats []Cat) *CatStore {
	if len(initialCats) == 0 {
		initialCats = basicCats
	}
	return &CatStore{cats: initialCats}
}

func (s *CatStore) getCatIndex(id string) int {
	return slices.IndexFunc(s.cats, func(c Cat) bool { return c.ID == id })
}

func (s *CatStore) Create(_ *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	newCat := Cat{ID: data.ID, Name: data.Object.(map[string]any)["name"].(string)}
	s.mu.Lock()
	s.cats = append(s.cats, newCat)
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: newCat.ID, Object: Cat{Name: newCat.Name}}
	s.broadcast(types.APIEvent{Name: types.CreateAPIEvent, Object: result})
	return result, nil
}

func (s *CatStore) Update(_ *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	s.mu.Lock()
	i := s.getCatIndex(id)
	if i == -1 {
		s.mu.Unlock()
		return types.APIObject{}, validation.NotFound
	}
	newName := data.Object.(map[string]any)["name"].(string)
	s.cats[i].Name = newName
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: id, Object: Cat{Name: newName}}
	s.broadcast(types.APIEvent{Name: types.ChangeAPIEvent, Object: result})
	return result, nil
}

func (s *CatStore) Delete(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	s.mu.Lock()
	i := s.getCatIndex(id)
	if i == -1 {
		s.mu.Unlock()
		return types.APIObject{}, validation.NotFound
	}
	cat := s.cats[i]
	s.cats = slices.Delete(s.cats, i, i+1)
	s.mu.Unlock()
	result := types.APIObject{Type: schema.ID, ID: id, Object: Cat{Name: cat.Name}}
	s.broadcast(types.APIEvent{Name: types.RemoveAPIEvent, Object: result})
	return result, nil
}

func (s *CatStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	objects := make([]types.APIObject, len(s.cats))
	for i, cat := range s.cats {
		objects[i] = types.APIObject{Type: schema.ID, ID: cat.ID, Object: Cat{Name: cat.Name}}
	}
	return types.APIObjectList{Objects: objects}, nil
}

func (s *CatStore) ByID(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	i := s.getCatIndex(id)
	if i == -1 {
		return types.APIObject{}, validation.NotFound
	}
	return types.APIObject{Type: schema.ID, ID: id, Object: Cat{Name: s.cats[i].Name}}, nil
}

// --- helpers ---

// newMultiPrefixRouter builds a ServeMux that routes /{prefix}/{type} and
// /{prefix}/{type}/{name} to s, matching the pattern from example.go.
func newMultiPrefixRouter(s *server.Server) *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/{prefix}/{type}", s)
	router.Handle("/{prefix}/{type}/{name}", s)
	return router
}

// mustDecodeList decodes a JSON collection response and returns the "data" slice.
func mustDecodeList(t *testing.T, resp *http.Response) []interface{} {
	t.Helper()
	var body map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	items, ok := body["data"].([]interface{})
	require.True(t, ok, "response body should contain a 'data' array")
	return items
}

// mustDecodeLinks decodes a JSON response and returns the "links" map.
func mustDecodeLinks(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	links, ok := body["links"].(map[string]interface{})
	require.True(t, ok, "response body should contain a 'links' object")
	return links
}

// --- tests ---

// TestAPIRoot_MultipleSchemas_SingleVersion verifies that two resource types
// registered on the same APISchemas are both accessible under a single version
// prefix after calling apiroot.Register once.
func TestAPIRoot_MultipleSchemas_SingleVersion(t *testing.T) {
	s := server.DefaultAPIServer()
	s.Schemas.MustImportAndCustomize(Dog{}, func(schema *types.APISchema) {
		schema.Store = NewDogStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	s.Schemas.MustImportAndCustomize(Cat{}, func(schema *types.APISchema) {
		schema.Store = NewCatStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	apiroot.Register(s.Schemas, []string{"v1"})

	ts := httptest.NewServer(newMultiPrefixRouter(s))
	defer ts.Close()

	t.Run("list dogs", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/dogs")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		items := mustDecodeList(t, resp)
		assert.Len(t, items, len(basicDogs))
		for i, basicDog := range basicDogs {
			assert.Equal(t, basicDog.ID, items[i].(map[string]interface{})["id"])
			assert.Equal(t, basicDog.Name, items[i].(map[string]interface{})["name"])
		}
	})

	t.Run("list cats", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/cats")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		items := mustDecodeList(t, resp)
		assert.Len(t, items, len(basicCats))
		for i, basicCat := range basicCats {
			assert.Equal(t, basicCat.ID, items[i].(map[string]interface{})["id"])
			assert.Equal(t, basicCat.Name, items[i].(map[string]interface{})["name"])
		}
	})

	t.Run("get dog by id", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/dogs/pluto")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "pluto", body["id"])
	})

	t.Run("get cat by id", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/cats/felix")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "felix", body["id"])
	})
}

// TestAPIRoot_MultipleVersions_MultipleSchemas verifies that when
// apiroot.Register is called with multiple version strings, each resource type
// is accessible under every version prefix, both as a collection and by ID.
func TestAPIRoot_MultipleVersions_MultipleSchemas(t *testing.T) {
	s := server.DefaultAPIServer()
	s.Schemas.MustImportAndCustomize(Dog{}, func(schema *types.APISchema) {
		schema.Store = NewDogStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	s.Schemas.MustImportAndCustomize(Cat{}, func(schema *types.APISchema) {
		schema.Store = NewCatStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	apiroot.Register(s.Schemas, []string{"v1", "v2"})

	ts := httptest.NewServer(newMultiPrefixRouter(s))
	defer ts.Close()

	resources := []struct {
		plural   string
		sampleID string
		count    int
	}{
		{"dogs", "pluto", len(basicDogs)},
		{"cats", "felix", len(basicCats)},
	}

	for _, version := range []string{"v1", "v2"} {
		for _, r := range resources {
			version, r := version, r

			t.Run("list "+version+"/"+r.plural, func(t *testing.T) {
				resp, err := http.Get(ts.URL + "/" + version + "/" + r.plural)
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Len(t, mustDecodeList(t, resp), r.count)
			})

			t.Run("get "+version+"/"+r.plural+"/"+r.sampleID, func(t *testing.T) {
				resp, err := http.Get(ts.URL + "/" + version + "/" + r.plural + "/" + r.sampleID)
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
	}
}

// TestAPIRoot_ByID_LinksIncludeRegisteredSchemas verifies that fetching the
// apiRoot resource for a given version returns a "links" map that includes an
// entry for every registered collection. This confirms that apiroot.Register
// correctly surfaces all schema collections to API clients navigating via
// hypermedia links.
func TestAPIRoot_ByID_LinksIncludeRegisteredSchemas(t *testing.T) {
	s := server.DefaultAPIServer()
	s.Schemas.MustImportAndCustomize(Dog{}, func(schema *types.APISchema) {
		schema.Store = NewDogStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	s.Schemas.MustImportAndCustomize(Cat{}, func(schema *types.APISchema) {
		schema.Store = NewCatStore(nil)
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})
	apiroot.Register(s.Schemas, []string{"v1", "v2"})

	ts := httptest.NewServer(newMultiPrefixRouter(s))
	defer ts.Close()

	for _, version := range []string{"v1", "v2"} {
		version := version
		t.Run(version, func(t *testing.T) {
			resp, err := http.Get(ts.URL + "/" + version + "/apiRoot/" + version)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			links := mustDecodeLinks(t, resp)
			assert.Contains(t, links, "dogs", "apiRoot links should include dogs collection")
			assert.Contains(t, links, "cats", "apiRoot links should include cats collection")
		})
	}
}
