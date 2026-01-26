package parse

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResponseFormat(t *testing.T) {
	tests := []struct {
		name           string
		queryFormat    string
		acceptHeader   string
		userAgent      string
		expectedFormat string
	}{
		{
			name:           "explicit json format",
			queryFormat:    "json",
			expectedFormat: "json",
		},
		{
			name:           "explicit yaml format",
			queryFormat:    "yaml",
			expectedFormat: "yaml",
		},
		{
			name:           "explicit html format",
			queryFormat:    "html",
			expectedFormat: "html",
		},
		{
			name:           "explicit jsonl format",
			queryFormat:    "jsonl",
			expectedFormat: "jsonl",
		},
		{
			name:           "format with spaces",
			queryFormat:    "  JSON  ",
			expectedFormat: "json",
		},
		{
			name:           "invalid format defaults to json",
			queryFormat:    "xml",
			expectedFormat: "json",
		},
		{
			name:           "browser user agent",
			userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			acceptHeader:   "*/*",
			expectedFormat: "html",
		},
		{
			name:           "accept yaml header",
			acceptHeader:   "application/yaml",
			expectedFormat: "yaml",
		},
		{
			name:           "accept jsonl header",
			acceptHeader:   "application/jsonl",
			expectedFormat: "jsonl",
		},
		{
			name:           "default to json",
			expectedFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.queryFormat != "" {
				q := req.URL.Query()
				q.Set("_format", tt.queryFormat)
				req.URL.RawQuery = q.Encode()
			}

			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			format := parseResponseFormat(req)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

func TestIsYaml(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		expected     bool
	}{
		{
			name:         "yaml accept header",
			acceptHeader: "application/yaml",
			expected:     true,
		},
		{
			name:         "yaml with other types",
			acceptHeader: "text/html,application/yaml,application/json",
			expected:     true,
		},
		{
			name:         "json accept header",
			acceptHeader: "application/json",
			expected:     false,
		},
		{
			name:     "empty accept header",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			result := isYaml(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJSONL(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		expected     bool
	}{
		{
			name:         "jsonl accept header",
			acceptHeader: "application/jsonl",
			expected:     true,
		},
		{
			name:         "jsonl with other types",
			acceptHeader: "text/html,application/jsonl,application/json",
			expected:     true,
		},
		{
			name:         "json accept header",
			acceptHeader: "application/json",
			expected:     false,
		},
		{
			name:     "empty accept header",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			result := isJSONL(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name           string
		requestMethod  string
		queryMethod    string
		expectedMethod string
	}{
		{
			name:           "GET method from request",
			requestMethod:  "GET",
			expectedMethod: "GET",
		},
		{
			name:           "POST method from request",
			requestMethod:  "POST",
			expectedMethod: "POST",
		},
		{
			name:           "override with query param",
			requestMethod:  "POST",
			queryMethod:    "PUT",
			expectedMethod: "PUT",
		},
		{
			name:           "DELETE method from query",
			requestMethod:  "GET",
			queryMethod:    "DELETE",
			expectedMethod: "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.requestMethod, "/", nil)
			if tt.queryMethod != "" {
				q := req.URL.Query()
				q.Set("_method", tt.queryMethod)
				req.URL.RawQuery = q.Encode()
			}

			method := parseMethod(req)
			assert.Equal(t, tt.expectedMethod, method)
		})
	}
}

func TestValuesToBody(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected types.APIObject
	}{
		{
			name: "simple key-value",
			input: map[string][]string{
				"name": {"test"},
			},
			expected: types.APIObject{
				Object: map[string]interface{}{
					"name": []string{"test"},
				},
			},
		},
		{
			name: "multiple values",
			input: map[string][]string{
				"tags": {"tag1", "tag2", "tag3"},
			},
			expected: types.APIObject{
				Object: map[string]interface{}{
					"tags": []string{"tag1", "tag2", "tag3"},
				},
			},
		},
		{
			name: "multiple fields",
			input: map[string][]string{
				"name":  {"test"},
				"email": {"test@example.com"},
			},
			expected: types.APIObject{
				Object: map[string]interface{}{
					"name":  []string{"test"},
					"email": []string{"test@example.com"},
				},
			},
		},
		{
			name:  "empty input",
			input: map[string][]string{},
			expected: types.APIObject{
				Object: map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesToBody(tt.input)
			assert.Equal(t, tt.expected.Object, result.Object)
		})
	}
}

func TestParse(t *testing.T) {
	mockURLParser := func(rw http.ResponseWriter, req *http.Request, schemas *types.APISchemas) (ParsedURL, error) {
		return ParsedURL{
			Type:      "testtype",
			Name:      "testname",
			Namespace: "testnamespace",
			Link:      "",
			Method:    "GET",
			Action:    "",
			Prefix:    "/v1",
			Query:     url.Values{},
		}, nil
	}

	t.Run("parse with nil request", func(t *testing.T) {
		apiOp := &types.APIRequest{
			Schemas: &types.APISchemas{},
		}

		err := Parse(apiOp, mockURLParser)
		require.NoError(t, err)
		assert.NotNil(t, apiOp.Request)
		assert.Equal(t, "GET", apiOp.Method)
	})

	t.Run("parse with existing request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		rw := httptest.NewRecorder()

		apiOp := &types.APIRequest{
			Request:  req,
			Response: rw,
			Schemas:  &types.APISchemas{},
		}

		err := Parse(apiOp, mockURLParser)
		require.NoError(t, err)
		assert.Equal(t, "POST", apiOp.Method)
		assert.Equal(t, "testtype", apiOp.Type)
		assert.Equal(t, "testname", apiOp.Name)
		assert.Equal(t, "testnamespace", apiOp.Namespace)
		assert.Equal(t, "/v1", apiOp.URLPrefix)
		assert.NotNil(t, apiOp.URLBuilder)
	})

	t.Run("parse with format query param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?_format=yaml", nil)
		rw := httptest.NewRecorder()

		apiOp := &types.APIRequest{
			Request:  req,
			Response: rw,
			Schemas:  &types.APISchemas{},
		}

		err := Parse(apiOp, mockURLParser)
		require.NoError(t, err)
		assert.Equal(t, "yaml", apiOp.ResponseFormat)
	})

	t.Run("parse with method override", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test?_method=PUT", nil)
		rw := httptest.NewRecorder()

		apiOp := &types.APIRequest{
			Request:  req,
			Response: rw,
			Schemas:  &types.APISchemas{},
		}

		err := Parse(apiOp, mockURLParser)
		require.NoError(t, err)
		assert.Equal(t, "PUT", apiOp.Method)
	})

	t.Run("parse preserves existing apiOp values", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rw := httptest.NewRecorder()

		apiOp := &types.APIRequest{
			Request:        req,
			Response:       rw,
			Schemas:        &types.APISchemas{},
			Type:           "customtype",
			Name:           "customname",
			ResponseFormat: "html",
		}

		err := Parse(apiOp, mockURLParser)
		require.NoError(t, err)
		// Should preserve custom values
		assert.Equal(t, "customtype", apiOp.Type)
		assert.Equal(t, "customname", apiOp.Name)
		assert.Equal(t, "html", apiOp.ResponseFormat)
	})
}

func TestParsedURL(t *testing.T) {
	t.Run("ParsedURL struct creation", func(t *testing.T) {
		parsedURL := ParsedURL{
			Type:      "pod",
			Name:      "nginx",
			Namespace: "default",
			Link:      "self",
			Method:    "GET",
			Action:    "start",
			Prefix:    "/v1",
			SubContext: map[string]string{
				"key": "value",
			},
			Query: url.Values{
				"filter": []string{"name=test"},
			},
		}

		assert.Equal(t, "pod", parsedURL.Type)
		assert.Equal(t, "nginx", parsedURL.Name)
		assert.Equal(t, "default", parsedURL.Namespace)
		assert.Equal(t, "self", parsedURL.Link)
		assert.Equal(t, "GET", parsedURL.Method)
		assert.Equal(t, "start", parsedURL.Action)
		assert.Equal(t, "/v1", parsedURL.Prefix)
		assert.Equal(t, "value", parsedURL.SubContext["key"])
		assert.Equal(t, "name=test", parsedURL.Query.Get("filter"))
	})
}

func TestAllowedFormats(t *testing.T) {
	t.Run("verify allowed formats", func(t *testing.T) {
		assert.True(t, allowedFormats["html"])
		assert.True(t, allowedFormats["json"])
		assert.True(t, allowedFormats["jsonl"])
		assert.True(t, allowedFormats["yaml"])
		assert.False(t, allowedFormats["xml"])
		assert.False(t, allowedFormats["csv"])
	})
}
