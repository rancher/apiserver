package parse

import (
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
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
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
