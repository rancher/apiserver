package server

import (
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func newAccessRequest(collectionMethods, resourceMethods []string) *types.APIRequest {
	s := types.EmptyAPISchemas().MustAddSchema(types.APISchema{
		Schema: &schemas.Schema{
			ID:                "widgets",
			PluralName:        "widgets",
			CollectionMethods: collectionMethods,
			ResourceMethods:   resourceMethods,
		},
	})

	return &types.APIRequest{Schemas: s}
}

func TestSchemaBasedAccess_RejectsUnknownResource(t *testing.T) {
	access := &SchemaBasedAccess{}
	apiOp := newAccessRequest([]string{http.MethodGet}, []string{http.MethodGet})

	err := access.CanDo(apiOp, "unknown", http.MethodGet, "namespace", "name")
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func TestSchemaBasedAccessCanDoWhenMethodIsInMethodList(t *testing.T) {
	access := &SchemaBasedAccess{}
	tests := []struct {
		name              string
		verb              string
		collectionMethods []string
		resourceMethods   []string
	}{
		{
			name:              "allows get when collection supports get",
			verb:              http.MethodGet,
			collectionMethods: []string{http.MethodGet},
		},
		{
			name:              "allows post when collection supports post",
			verb:              http.MethodPost,
			collectionMethods: []string{http.MethodPost},
		},
		{
			name:            "allows put when resource supports put",
			verb:            http.MethodPut,
			resourceMethods: []string{http.MethodPut},
		},
		{
			name:            "allows patch when resource supports patch",
			verb:            http.MethodPatch,
			resourceMethods: []string{http.MethodPatch},
		},
		{
			name:            "allows delete when resource supports delete",
			verb:            http.MethodDelete,
			resourceMethods: []string{http.MethodDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiOp := newAccessRequest(tt.collectionMethods, tt.resourceMethods)
			err := access.CanDo(apiOp, "widgets", tt.verb, "default", "one")
			require.NoError(t, err)
		})
	}
}

func TestSchemaBasedAccessCanDo_VariousMethods(t *testing.T) {
	type testCase struct {
		description       string
		requestMethod     string
		collectionMethods []string
		resourceMethods   []string
		expectedErr       bool
		expectedErrorCode *validation.ErrorCode
	}
	var tests []testCase
	tests = append(tests, testCase{
		requestMethod:     "UnknownRequest",
		collectionMethods: []string{http.MethodGet},
		resourceMethods:   []string{http.MethodGet},
		expectedErr:       true,
		expectedErrorCode: &validation.PermissionDenied,
	})
	tests = append(tests, testCase{
		description:       "Rejects an unsupported verb",
		requestMethod:     http.MethodOptions,
		collectionMethods: []string{http.MethodGet},
		resourceMethods:   []string{http.MethodGet},
		expectedErr:       true,
		expectedErrorCode: &validation.PermissionDenied,
	})
	tests = append(tests, testCase{
		description:       "Can list when collection supports post",
		requestMethod:     http.MethodGet,
		collectionMethods: []string{http.MethodPost},
	})
	tests = append(tests, testCase{
		description:       "Can list when collection supports get",
		requestMethod:     http.MethodGet,
		collectionMethods: []string{http.MethodGet},
	})
	tests = append(tests, testCase{
		description:   "Can't list when collection doesn't do get or post",
		requestMethod: http.MethodGet,
		expectedErr:   true,
	})
	tests = append(tests, testCase{
		description:     "Can update when resource methods supports put",
		requestMethod:   http.MethodPut,
		resourceMethods: []string{http.MethodPut},
	})
	tests = append(tests, testCase{
		description:       "Can't update when resource methods don't support put",
		requestMethod:     http.MethodPut,
		collectionMethods: []string{http.MethodPut},
		resourceMethods:   []string{http.MethodPost},
		expectedErr:       true,
	})
	tests = append(tests, testCase{
		description:     "Can patch when resource methods supports Patch",
		requestMethod:   http.MethodPatch,
		resourceMethods: []string{http.MethodPatch},
	})
	tests = append(tests, testCase{
		description:       "Can't patch when resource methods don't support Patch",
		requestMethod:     http.MethodPatch,
		collectionMethods: []string{http.MethodPatch},
		resourceMethods:   []string{http.MethodPost},
		expectedErr:       true,
	})
	tests = append(tests, testCase{
		description:     "Can delete when resource methods supports put",
		requestMethod:   http.MethodDelete,
		resourceMethods: []string{http.MethodDelete},
	})
	tests = append(tests, testCase{
		description:       "Can't delete when resource methods don't support put",
		requestMethod:     http.MethodDelete,
		collectionMethods: []string{http.MethodDelete},
		resourceMethods:   []string{http.MethodPost},
		expectedErr:       true,
	})
	tests = append(tests, testCase{
		description:       "Can create when collection methods supports put",
		requestMethod:     http.MethodPost,
		collectionMethods: []string{http.MethodPost},
	})
	tests = append(tests, testCase{
		description:   "Can't create when collection doesn't do get or post",
		requestMethod: http.MethodPost,
		expectedErr:   true,
	})
	tests = append(tests, testCase{
		description:     "Can't create when collection doesn't do post, even when resource-methods do",
		requestMethod:   http.MethodPost,
		resourceMethods: []string{http.MethodPost},
		expectedErr:     true,
	})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			access := &SchemaBasedAccess{}
			apiOp := newAccessRequest(test.collectionMethods, test.resourceMethods)
			err := access.CanDo(apiOp, "widgets", test.requestMethod, "theNamespace", "theName")
			if test.expectedErr {
				require.Error(t, err)
				if test.expectedErrorCode != nil {
					apiErr, ok := err.(*apierror.APIError)
					require.True(t, ok)
					require.Equal(t, apiErr.Code.Code, validation.PermissionDenied.Code)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
