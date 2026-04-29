package parse

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func schemaWithMethods(collection, resource []string) *types.APISchema {
	return &types.APISchema{
		Schema: &schemas.Schema{
			CollectionMethods: collection,
			ResourceMethods:   resource,
		},
	}
}

func TestValidate(t *testing.T) {
	type testCase struct {
		description string
		method      string
		action      string
		name        string
		requestType string
		link        string
		collection  []string
		resource    []string

		expectedErr       bool
		expectedErrorCode *validation.ErrorCode
	}
	var tests []testCase
	tests = append(tests, testCase{
		description: "Allow Post with a non-empty action",
		method:      http.MethodPost,
		action:      "deploy",
		requestType: "widgets",
	})
	tests = append(tests, testCase{
		description: "Allow Post with a non-empty action even with a conflicting schema",
		method:      http.MethodPost,
		action:      "deploy",
		requestType: "widgets",
		collection:  []string{http.MethodGet},
	})

	tests = append(tests, testCase{
		description:       "Unsupported method not supported",
		method:            http.MethodOptions,
		expectedErr:       true,
		expectedErrorCode: &validation.MethodNotAllowed,
	})
	tests = append(tests, testCase{
		description: "Allow Post with an empty type",
		method:      http.MethodPost,
	})
	tests = append(tests, testCase{
		description: "Allow Post with an empty type even with a conflicting schema",
		method:      http.MethodPost,
		collection:  []string{http.MethodGet},
	})
	tests = append(tests, testCase{
		description: "Allow Post with an empty schema type",
		method:      http.MethodPost,
		requestType: "widgets",
	})
	tests = append(tests, testCase{
		description: "Allow Post with a non-empty link",
		method:      http.MethodPost,
		requestType: "widgets",
		collection:  []string{http.MethodGet},
		link:        "log",
	})
	tests = append(tests, testCase{
		description: "Delete allowed if in resource-methods",
		method:      http.MethodDelete,
		action:      "deploy",
		requestType: "widgets",
		name:        "useResourceMethods",
		resource:    []string{http.MethodDelete},
		collection:  []string{http.MethodPut},
	})
	tests = append(tests, testCase{
		description: "Delete allowed if in collection",
		method:      http.MethodDelete,
		action:      "deploy",
		requestType: "widgets",
		collection:  []string{http.MethodDelete},
	})
	tests = append(tests, testCase{
		description:       "Get not allowed if post in resource-methods",
		method:            http.MethodGet,
		action:            "deploy",
		requestType:       "widgets",
		name:              "useResourceMethods",
		resource:          []string{http.MethodPost},
		expectedErr:       true,
		expectedErrorCode: &validation.PermissionDenied,
	})
	tests = append(tests, testCase{
		description: "Get allowed if post in schema collection",
		method:      http.MethodGet,
		action:      "deploy",
		requestType: "widgets",
		collection:  []string{http.MethodPost},
	})
	tests = append(tests, testCase{
		description:       "Delete not allowed if schema is given and it doesn't appear in resourceMethods",
		method:            http.MethodDelete,
		action:            "deploy",
		requestType:       "widgets",
		resource:          []string{http.MethodPut},
		collection:        []string{http.MethodDelete},
		name:              "useResourceMethods",
		expectedErr:       true,
		expectedErrorCode: &validation.PermissionDenied,
	})
	tests = append(tests, testCase{
		description:       "Delete not allowed if schema is given and it doesn't appear in collection",
		method:            http.MethodDelete,
		action:            "deploy",
		requestType:       "widgets",
		resource:          []string{http.MethodDelete},
		collection:        []string{http.MethodPut},
		expectedErr:       true,
		expectedErrorCode: &validation.PermissionDenied,
	})
	t.Parallel()
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := &types.APIRequest{
				Method: test.method,
				Action: test.action,
				Type:   test.requestType,
				Link:   test.link,
				Name:   test.name,
			}
			if len(test.collection) > 0 || len(test.resource) > 0 {
				req.Schema = schemaWithMethods(test.collection, test.resource)
			}
			if test.description == "Delete allowed if in collection" {
				fmt.Println("stop here")
			}
			err := ValidateMethod(req)
			if test.expectedErr {
				require.Error(t, err)
				if test.expectedErrorCode != nil {
					apiErr, ok := err.(*apierror.APIError)
					require.True(t, ok)
					require.Equal(t, test.expectedErrorCode.Code, apiErr.Code.Code)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func hideTestValidateMethodAllowsActionPostWithoutSchemaCheck(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodPost,
		Action: "deploy",
		Type:   "widgets",
		Schema: schemaWithMethods(nil, nil),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodActionShortcutDoesNotApplyForNonPostMethod(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodDelete,
		Action: "deploy",
		Type:   "widgets",
		Schema: schemaWithMethods(nil, []string{http.MethodGet}),
	}
	err := ValidateMethod(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func hideTestValidateMethodRejectsUnsupportedHTTPMethod(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodOptions,
	}
	err := ValidateMethod(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.MethodNotAllowed, apiErr.Code)
}

func hideTestValidateMethodPassesWhenTypeIsEmpty(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodGet,
		Type:   "",
		Schema: schemaWithMethods(nil, nil),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodPassesWhenSchemaIsNil(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodDelete,
		Type:   "widgets",
		Schema: nil,
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodPassesWhenLinkIsSet(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodPatch,
		Type:   "widgets",
		Link:   "log",
		Schema: schemaWithMethods(nil, nil),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodAllowsCollectionMethodForRequestWithNoName(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodPost,
		Type:   "widgets",
		Name:   "",
		Schema: schemaWithMethods([]string{http.MethodPost}, nil),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodAllowsResourceMethodForRequestWithName(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodPut,
		Type:   "widgets",
		Name:   "one",
		Schema: schemaWithMethods(nil, []string{http.MethodPut}),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodAllowsGetOnCollectionWhenPostIsListed(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodGet,
		Type:   "widgets",
		Name:   "",
		Schema: schemaWithMethods([]string{http.MethodPost}, nil),
	}
	require.NoError(t, ValidateMethod(req))
}

func hideTestValidateMethodRejectsMethodNotInCollectionMethods(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodDelete,
		Type:   "widgets",
		Name:   "",
		Schema: schemaWithMethods([]string{http.MethodGet}, nil),
	}
	err := ValidateMethod(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func hideTestValidateMethodRejectsMethodNotInResourceMethods(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodPatch,
		Type:   "widgets",
		Name:   "one",
		Schema: schemaWithMethods(nil, []string{http.MethodGet}),
	}
	err := ValidateMethod(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func hideTestValidateMethodDoesNotAllowGetOnCollectionWhenPostListedButNameIsSet(t *testing.T) {
	req := &types.APIRequest{
		Method: http.MethodGet,
		Type:   "widgets",
		Name:   "one",
		Schema: schemaWithMethods([]string{http.MethodPost}, []string{http.MethodPost}),
	}
	err := ValidateMethod(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}
