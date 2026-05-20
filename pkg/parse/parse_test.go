package parse

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func TestParsePopulatesRequestFieldsFromURLParser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/widgets/one?_method=PUT", nil)
	apiReq := &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
	}

	parser := func(http.ResponseWriter, *http.Request, *types.APISchemas) (ParsedURL, error) {
		return ParsedURL{
			Type:      "widgets",
			Name:      "one",
			Namespace: "default",
			Link:      "details",
			Action:    "do",
			Prefix:    "v1",
			Query:     url.Values{"q": []string{"v"}},
		}, nil
	}

	err := Parse(apiReq, parser)
	require.NoError(t, err)
	require.Equal(t, "widgets", apiReq.Type)
	require.Equal(t, "one", apiReq.Name)
	require.Equal(t, "default", apiReq.Namespace)
	require.Equal(t, "details", apiReq.Link)
	require.Equal(t, "do", apiReq.Action)
	require.Equal(t, "v1", apiReq.URLPrefix)
	require.Equal(t, "PUT", apiReq.Method)
	require.Equal(t, "v", apiReq.Query.Get("q"))
	require.NotNil(t, apiReq.URLBuilder)
	require.Equal(t, apiReq, types.GetAPIContext(apiReq.Context()))
}

func TestParsePreservesPrepopulatedFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/widgets/two", nil)
	apiReq := &types.APIRequest{
		Request:        req,
		Response:       httptest.NewRecorder(),
		Type:           "existingType",
		Name:           "existingName",
		Namespace:      "existingNamespace",
		Link:           "existingLink",
		Action:         "existingAction",
		URLPrefix:      "existingPrefix",
		Method:         http.MethodDelete,
		ResponseFormat: "yaml",
		Query:          url.Values{"existing": []string{"1"}},
	}

	parser := func(http.ResponseWriter, *http.Request, *types.APISchemas) (ParsedURL, error) {
		return ParsedURL{
			Type:      "newType",
			Name:      "newName",
			Namespace: "newNamespace",
			Link:      "newLink",
			Action:    "newAction",
			Prefix:    "newPrefix",
			Method:    http.MethodPost,
			Query:     url.Values{"new": []string{"2"}},
		}, nil
	}

	err := Parse(apiReq, parser)
	require.NoError(t, err)
	require.Equal(t, "existingType", apiReq.Type)
	require.Equal(t, "existingName", apiReq.Name)
	require.Equal(t, "existingNamespace", apiReq.Namespace)
	require.Equal(t, "existingLink", apiReq.Link)
	require.Equal(t, "existingAction", apiReq.Action)
	require.Equal(t, "existingPrefix", apiReq.URLPrefix)
	require.Equal(t, http.MethodDelete, apiReq.Method)
	require.Equal(t, "yaml", apiReq.ResponseFormat)
	require.Equal(t, "1", apiReq.Query.Get("existing"))
	require.Equal(t, "", apiReq.Query.Get("new"))
}

func TestParseReturnsURLParserErrorAfterSettingDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/widgets", nil)
	apiReq := &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
	}
	parseErr := errors.New("bad parse")

	parser := func(http.ResponseWriter, *http.Request, *types.APISchemas) (ParsedURL, error) {
		return ParsedURL{Type: "widgets", Query: url.Values{"k": []string{"v"}}}, parseErr
	}

	err := Parse(apiReq, parser)
	require.ErrorIs(t, err, parseErr)
	require.Equal(t, http.MethodGet, apiReq.Method)
	require.Equal(t, "json", apiReq.ResponseFormat)
	require.Equal(t, "widgets", apiReq.Type)
	require.Equal(t, "v", apiReq.Query.Get("k"))
}

func TestParseUsesSchemaLookupAndNormalizesTypeToSchemaID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/widgets", nil)
	s := types.EmptyAPISchemas().MustAddSchema(types.APISchema{
		Schema: &schemas.Schema{
			ID:                "widget",
			PluralName:        "widgets",
			CollectionMethods: []string{http.MethodGet},
			ResourceMethods:   []string{http.MethodGet},
		},
	})

	apiReq := &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
		Schemas:  s,
	}

	parser := func(http.ResponseWriter, *http.Request, *types.APISchemas) (ParsedURL, error) {
		return ParsedURL{Type: "widgets"}, nil
	}

	err := Parse(apiReq, parser)
	require.NoError(t, err)
	require.NotNil(t, apiReq.Schema)
	require.Equal(t, "widget", apiReq.Type)
}

func TestParseResponseFormat(t *testing.T) {
	type testCase struct {
		description     string
		userAgent       string
		accept          string
		formatParameter string
		expectedFormat  string
	}
	var tests []testCase
	tests = append(tests, testCase{
		description:     "Uses explicit format when allowed",
		formatParameter: "/?_format=%20%20Yaml%20%20",
		expectedFormat:  "yaml"})
	tests = append(tests, testCase{
		description:    "Returns html for browser request",
		userAgent:      "Mozilla/5.0",
		accept:         "*/*",
		expectedFormat: "html"})
	tests = append(tests, testCase{
		description:    "Returns yaml when accepted and not browser",
		userAgent:      "curl/5.0",
		accept:         "application/yaml",
		expectedFormat: "yaml"})
	tests = append(tests, testCase{
		description:     "Returns json when format is unknown",
		formatParameter: "/?_format=xml",
		userAgent:       "curl/5.0",
		accept:          "application/json",
		expectedFormat:  "json"})
	tests = append(tests, testCase{
		description:    "Returns jsonl when accepted and not browser",
		userAgent:      "curl/5.0",
		accept:         "application/jsonl",
		expectedFormat: "jsonl"})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			queryPath := test.formatParameter
			if queryPath == "" {
				queryPath = "/"
			}
			req := httptest.NewRequest(http.MethodGet, queryPath, nil)
			req.Header.Set("User-Agent", test.userAgent)
			req.Header.Set("Accept", test.accept)
			result := parseResponseFormat(req)
			require.Equal(t, test.expectedFormat, result)
		})
	}
}

func TestParseMethodUsesMethodOverrideWhenProvided(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?_method=DELETE", nil)
	require.Equal(t, http.MethodDelete, parseMethod(req))
}

func TestParseMethodFallsBackToRequestMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/", nil)
	require.Equal(t, http.MethodPatch, parseMethod(req))
}

func TestBodyReturnsMultipartValuesWhenMultipartFormExists(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.MultipartForm = &multipart.Form{Value: map[string][]string{"name": {"alice", "bob"}}}

	body, err := Body(req)
	require.NoError(t, err)

	payload, ok := body.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, []string{"alice", "bob"}, payload["name"])
}

func TestBodyReturnsFormValuesWhenPostFormExists(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.PostForm = url.Values{"key": []string{"v1", "v2"}}
	req.Form = url.Values{"key": []string{"v1", "v2"}}

	body, err := Body(req)
	require.NoError(t, err)

	payload, ok := body.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, []string{"v1", "v2"}, payload["key"])
}

func TestBodyParsesJSONWhenNoFormDataProvided(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":"widget","id":"id-1","name":"test"}`))
	req.Header.Set("Content-type", "application/json")

	body, err := Body(req)
	require.NoError(t, err)
	require.Equal(t, "widget", body.Type)
	require.Equal(t, "id-1", body.ID)
}

func TestBodyReturnsErrorForInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))
	req.Header.Set("Content-type", "application/json")

	_, err := Body(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
}

func TestBodyReturnsEmptyObjectForMethodsWithoutBodyParsing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"type":"widget"}`))
	body, err := Body(req)
	require.NoError(t, err)
	require.Equal(t, types.APIObject{}, body)
}
