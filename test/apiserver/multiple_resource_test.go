package apiserver_test

import (
	//"bytes"
	"encoding/json"
	"github.com/rancher/apiserver/pkg/store/apiroot"
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

type Dog struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type Cat struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type CatStore struct {
	empty.Store
	cats []Cat
}

type DogStore struct {
	empty.Store
	dogs []Dog
}

var basicDogs = []Dog{
	{
		ID:   "pluto",
		Name: "disney",
	},
	{
		ID:   "krypto",
		Name: "dc",
	},
}

func NewDogStore(initialDogs []Dog) *DogStore {
	if len(initialDogs) == 0 {
		initialDogs = basicDogs
	}
	return &DogStore{dogs: initialDogs}
}

var basicCats = []Cat{
	{
		ID:   "felix",
		Name: "dell",
	},
	{
		ID:   "fritz",
		Name: "zap",
	},
}

func NewCatStore(initialCats []Cat) *CatStore {
	if len(initialCats) == 0 {
		initialCats = basicCats
	}
	return &CatStore{cats: initialCats}
}

func (s *DogStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	var returnedObjects []types.APIObject
	for _, dog := range s.dogs {
		returnedObjects = append(returnedObjects, types.APIObject{
			Type: schema.ID,
			ID:   dog.ID,
			Object: Dog{
				Name: dog.Name,
			},
		})
	}
	return types.APIObjectList{Objects: returnedObjects}, nil
}

func (s *DogStore) getDogIndex(id string) int {
	return slices.IndexFunc(s.dogs, func(d Dog) bool {
		return d.ID == id
	})
}

func (s *DogStore) ByID(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getDogIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	dog := s.dogs[index]
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Dog{
			Name: dog.Name,
		},
	}, nil
}

func (s *DogStore) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getDogIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	dogToDelete := s.dogs[index]
	s.dogs = slices.Delete(s.dogs, index, index+1)
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Dog{
			Name: dogToDelete.Name,
		},
	}, nil
}

func (s *DogStore) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	s.dogs = append(s.dogs, Dog{ID: data.ID, Name: data.Object.(map[string]any)["name"].(string)})
	return data, nil
}

func (s *DogStore) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	index := s.getDogIndex(id)
	if index == -1 {
		return data, validation.NotFound
	}
	newName, ok := data.Object.(map[string]any)["name"].(string)
	if !ok {
		return data, validation.NotFound
	}
	s.dogs[index].Name = newName
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Dog{
			Name: newName,
		},
	}, nil

}

func (s *CatStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	var returnedObjects []types.APIObject
	for _, cat := range s.cats {
		returnedObjects = append(returnedObjects, types.APIObject{
			Type: schema.ID,
			ID:   cat.ID,
			Object: Cat{
				Name: cat.Name,
			},
		})
	}
	return types.APIObjectList{Objects: returnedObjects}, nil
}

func (s *CatStore) getCatIndex(id string) int {
	return slices.IndexFunc(s.cats, func(d Cat) bool {
		return d.ID == id
	})
}

func (s *CatStore) ByID(_ *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getCatIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	cat := s.cats[index]
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Cat{
			Name: cat.Name,
		},
	}, nil
}

func (s *CatStore) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	index := s.getCatIndex(id)
	if index == -1 {
		return types.APIObject{}, validation.NotFound
	}
	catToDelete := s.cats[index]
	s.cats = slices.Delete(s.cats, index, index+1)
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Cat{
			Name: catToDelete.Name,
		},
	}, nil
}

func (s *CatStore) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	s.cats = append(s.cats, Cat{ID: data.ID, Name: data.Object.(map[string]any)["name"].(string)})
	return data, nil
}

func (s *CatStore) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	index := s.getCatIndex(id)
	if index == -1 {
		return data, validation.NotFound
	}
	newName, ok := data.Object.(map[string]any)["name"].(string)
	if !ok {
		return data, validation.NotFound
	}
	s.cats[index].Name = newName
	return types.APIObject{
		Type: schema.ID,
		ID:   id,
		Object: Cat{
			Name: newName,
		},
	}, nil
}

func TestBothResources_ListWorks(t *testing.T) {
	s := server.DefaultAPIServer()

	rootSchemas := types.EmptyAPISchemas()

	// ---- V3 (Dogs) ----
	v3Schemas := types.EmptyAPISchemas()
	dogStore := NewDogStore(nil)
	v3Schemas.MustImportAndCustomize(Dog{}, func(s *types.APISchema) {
		s.Store = dogStore
	})

	// ---- V4 (Cats) ----
	v4Schemas := types.EmptyAPISchemas()
	catStore := NewCatStore(nil)
	v4Schemas.MustImportAndCustomize(Cat{}, func(s *types.APISchema) {
		s.Store = catStore
	})

	// ---- Mount API roots ----
	apiroot.Register(rootSchemas, []string{"/v3"}, "dogs")
	apiroot.Register(rootSchemas, []string{"/v4"}, "cats")

	s.Schemas = rootSchemas
	ts := httptest.NewServer(s)
	defer ts.Close()

	// Verify neither is on v1
	for _, pet := range []string{"cats", "dogs"} {
		func() {
			resp, err := http.Get(ts.URL + "/v1/" + pet)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		}()
	}

	// Verify we can get dogs on /v3
	func() {
		resp, err := http.Get(ts.URL + "/v3/dogs")
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

		//require.Equal(t, "pluto", items[0].ID)
		//require.Equal(t, "krypto", items[1].ID)
	}()
}
