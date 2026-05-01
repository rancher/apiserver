package schema

import (
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/stretchr/testify/require"
)

func newTestSchema(id string, collection, resource []string) *types.APISchema {
	return &types.APISchema{
		Schema: &schemas.Schema{
			ID:                id,
			PluralName:        id + "s",
			CollectionMethods: collection,
			ResourceMethods:   resource,
		},
	}
}

func newTestRequest(s map[string]*types.APISchema) *types.APIRequest {
	return &types.APIRequest{
		Schemas: &types.APISchemas{
			Schemas: s,
		},
	}
}

func TestByID(t *testing.T) {
	type testCase struct {
		description      string
		resourceToSearch string

		expectedErr bool
	}
	var tests []testCase
	tests = append(tests, testCase{
		description:      "Returns schema when found",
		resourceToSearch: "widget",
	})
	tests = append(tests, testCase{
		description:      "Returns error when not found",
		resourceToSearch: "thingamajig",
		expectedErr:      true,
	})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			testResource := "widget"
			schema := newTestSchema(testResource, []string{http.MethodGet}, []string{http.MethodGet})
			schemaMap := map[string]*types.APISchema{testResource: schema}
			apiOp := newTestRequest(schemaMap)
			store := NewSchemaStore()

			obj, err := store.ByID(apiOp, nil, test.resourceToSearch)
			if test.expectedErr {
				require.Error(t, err)
				apiErr, ok := err.(*apierror.APIError)
				require.True(t, ok)
				require.Equal(t, validation.NotFound.Code, apiErr.Code.Code)
			} else {
				require.NoError(t, err)
				require.Equal(t, "schema", obj.Type)
				require.Equal(t, testResource, obj.ID)
			}
		})
	}
}

func TestListReturnsOnlyFilterableSchemasWithMethods(t *testing.T) {
	withMethods := newTestSchema("widgets", []string{http.MethodGet}, []string{http.MethodGet})
	withoutMethods := newTestSchema("internal", nil, nil)
	schemaMap := map[string]*types.APISchema{
		"widgets":  withMethods,
		"internal": withoutMethods,
	}
	apiOp := newTestRequest(schemaMap)
	store := NewSchemaStore()

	list, err := store.List(apiOp, nil)
	require.NoError(t, err)
	require.Len(t, list.Objects, 1)
	require.Equal(t, "widgets", list.Objects[0].ID)
}

func TestToAPIObjectRemovesAccessAttribute(t *testing.T) {
	schema := newTestSchema("widget", []string{http.MethodGet}, nil)
	schema.Attributes = map[string]interface{}{
		"access":  "secret",
		"visible": "yes",
	}

	obj := toAPIObject(schema)
	payload, ok := obj.Object.(*types.APISchema)
	require.True(t, ok)
	require.NotContains(t, payload.Attributes, "access")
	require.Contains(t, payload.Attributes, "visible")
}

func TestFilterSchemas(t *testing.T) {
	type testCase struct {
		description  string
		getSchemaMap func() map[string]*types.APISchema
		expectedIDs  []string
	}
	var tests []testCase
	tests = append(tests, testCase{
		description: "Omit schemas without methods",
		getSchemaMap: func() map[string]*types.APISchema {
			withMethods := newTestSchema("have", []string{http.MethodGet}, nil)
			withoutMethods := newTestSchema("without", nil, nil)
			return map[string]*types.APISchema{
				"have":    withMethods,
				"without": withoutMethods,
			}
		},
		expectedIDs: []string{"have"},
	})
	tests = append(tests, testCase{
		description: "Ignore duplicate IDs",
		getSchemaMap: func() map[string]*types.APISchema {
			withMethods := newTestSchema("have", []string{http.MethodGet}, nil)
			withoutMethods := newTestSchema("without", nil, nil)
			duplicateMethods := newTestSchema("have", []string{http.MethodPut}, nil)
			schemaMap := map[string]*types.APISchema{
				"have":      withMethods,
				"without":   withoutMethods,
				"duplicate": duplicateMethods,
			}
			return schemaMap
		},
		expectedIDs: []string{"have"},
	})
	tests = append(tests, testCase{
		description: "Include referenced schemas declared in resource fields",
		getSchemaMap: func() map[string]*types.APISchema {
			childSchema := newTestSchema("child", []string{http.MethodGet}, nil)
			parentSchema := newTestSchema("parent", []string{http.MethodGet}, nil)
			parentSchema.ResourceFields = map[string]schemas.Field{
				"ref": {Type: "child"},
			}

			schemaMap := map[string]*types.APISchema{
				"parent": parentSchema,
				"child":  childSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"parent", "child"},
	})
	tests = append(tests, testCase{
		description: "Include referenced schemas frojm resource actions",
		getSchemaMap: func() map[string]*types.APISchema {
			resultSchema := newTestSchema("result", []string{http.MethodGet}, nil)
			parentSchema := newTestSchema("parent", []string{http.MethodGet}, []string{http.MethodPost})
			parentSchema.ResourceActions = map[string]schemas.Action{
				"deploy": {Output: "result"},
			}
			schemaMap := map[string]*types.APISchema{
				"parent": parentSchema,
				"result": resultSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"parent", "result"},
	})
	tests = append(tests, testCase{
		description: "Include referenced schemas from collection actions",
		getSchemaMap: func() map[string]*types.APISchema {
			inputSchema := newTestSchema("input", []string{http.MethodGet}, nil)
			parentSchema := newTestSchema("parent", []string{http.MethodGet, http.MethodPost}, nil)
			parentSchema.CollectionActions = map[string]schemas.Action{
				"bulk": {Input: "input"},
			}

			schemaMap := map[string]*types.APISchema{
				"parent": parentSchema,
				"input":  inputSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"parent", "input"},
	})
	tests = append(tests, testCase{
		description: "Include nested references",
		getSchemaMap: func() map[string]*types.APISchema {
			deepSchema := newTestSchema("deep", []string{http.MethodGet}, nil)
			midSchema := newTestSchema("mid", []string{http.MethodGet}, nil)
			midSchema.ResourceFields = map[string]schemas.Field{
				"ref": {Type: "deep"},
			}
			topSchema := newTestSchema("top", []string{http.MethodGet}, nil)
			topSchema.ResourceFields = map[string]schemas.Field{
				"ref": {Type: "mid"},
			}

			schemaMap := map[string]*types.APISchema{
				"top":  topSchema,
				"mid":  midSchema,
				"deep": deepSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"top", "mid", "deep"},
	})
	tests = append(tests, testCase{
		description: "Ignore empty action types",
		getSchemaMap: func() map[string]*types.APISchema {
			parentSchema := newTestSchema("parent", []string{http.MethodGet, http.MethodPost}, nil)
			parentSchema.ResourceActions = map[string]schemas.Action{
				"noref": {Output: "", Input: ""},
			}

			schemaMap := map[string]*types.APISchema{
				"parent": parentSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"parent"},
	})
	tests = append(tests, testCase{
		description: "Ignore complex array type",
		getSchemaMap: func() map[string]*types.APISchema {
			parentSchema := newTestSchema("parent", []string{http.MethodGet}, nil)
			parentSchema.ResourceFields = map[string]schemas.Field{
				"items": {Type: "array[widget]"},
			}

			schemaMap := map[string]*types.APISchema{
				"parent": parentSchema,
			}
			return schemaMap
		},
		expectedIDs: []string{"parent"},
	})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			schemaMap := test.getSchemaMap()
			apiOp := newTestRequest(schemaMap)

			list := FilterSchemas(apiOp, schemaMap)
			require.Len(t, list.Objects, len(test.expectedIDs))
			if len(list.Objects) == 1 {
				require.Equal(t, test.expectedIDs[0], list.Objects[0].ID)
			} else {
				ids := make(map[string]bool)
				for _, obj := range list.Objects {
					ids[obj.ID] = true
				}
				for _, id := range test.expectedIDs {
					require.True(t, ids[id])
				}
			}
		})
	}
}
