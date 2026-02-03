package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/apiroot"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
)

type Foo struct {
	Bar string `json:"bar"`
}

type FooStore struct {
	empty.Store
}

func (f *FooStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	return types.APIObject{
		Type: "foos",
		ID:   id,
		Object: Foo{
			Bar: "baz",
		},
	}, nil
}

func (f *FooStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	return types.APIObjectList{
		Objects: []types.APIObject{
			{
				Type: "foostore",
				ID:   "foo",
				Object: Foo{
					Bar: "baz",
				},
			},
		},
	}, nil
}

func main() {
	// Create the default server
	s := server.DefaultAPIServer()

	// Add some types to it and setup the store and supported methods
	s.Schemas.MustImportAndCustomize(Foo{}, func(schema *types.APISchema) {
		schema.Store = &FooStore{}
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})

	// Register root handler to list api versions
	apiroot.Register(s.Schemas, []string{"v1", "v2"})

	// Setup HTTP router with path parameters (Go 1.22+ pattern matching)
	router := http.NewServeMux()
	router.Handle("GET /{prefix}/{type}", s)
	router.Handle("GET /{prefix}/{type}/{name}", s)

	// When a route is found construct a custom API request to serves up the API root content
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		s.Handle(&types.APIRequest{
			Request:   r,
			Response:  rw,
			Type:      "apiRoot",
			URLPrefix: "v1",
		})
	})

	// Start API Server

	port := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	log.Printf("Listening on %s", port)
	log.Fatal(http.ListenAndServe(port, router))
}
