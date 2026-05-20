package parse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func TestReadBodyReturnsEmptyObjectForMethodsWithoutBodySupport(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"type":"widget","id":"one"}`))

	obj, err := ReadBody(req)
	require.NoError(t, err)
	require.Equal(t, "", obj.Type)
	require.Equal(t, "", obj.ID)
	require.Nil(t, obj.Object)
}

func TestReadBodyParsesJSONForPostRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":"widget","id":"one","name":"alpha"}`))
	req.Header.Set("Content-type", "application/json")

	obj, err := ReadBody(req)
	require.NoError(t, err)
	require.Equal(t, "widget", obj.Type)
	require.Equal(t, "one", obj.ID)

	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "alpha", payload["name"])
}

func TestReadBodyParsesYAMLWhenContentTypeIsApplicationYAML(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader("type: widget\nid: two\nname: beta\n"))
	req.Header.Set("Content-type", "application/yaml")

	obj, err := ReadBody(req)
	require.NoError(t, err)
	require.Equal(t, "widget", obj.Type)
	require.Equal(t, "two", obj.ID)

	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "beta", payload["name"])
}

func TestReadBodyReturnsInvalidBodyErrorForMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":`))
	req.Header.Set("Content-type", "application/json")

	_, err := ReadBody(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
	require.Contains(t, apiErr.Message, "Failed to parse body")
}

func TestReadBodyFallsBackToJSONDecoderWhenYAMLContentTypeHasCharset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("type: widget\nid: three\n"))
	req.Header.Set("Content-type", "application/yaml; charset=utf-8")

	_, err := ReadBody(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
}

func TestReadBodyReturnsInvalidBodyErrorForMalformedYAML(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("type:\nmissed\n  'extra indent here'\n"))
	req.Header.Set("Content-type", "application/yaml")

	_, err := ReadBody(req)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
	require.Contains(t, apiErr.Message, "Failed to parse body")
}

func TestReadBody_JSONNumbersAreRead(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":"widget","id":"four","count":12}`))
	req.Header.Set("Content-type", "application/json")

	obj, err := ReadBody(req)
	require.NoError(t, err)

	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)
	count, ok := payload["count"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "12", count.String())
}

func TestToAPIConvertsTypeAndIDToStrings(t *testing.T) {
	data := map[string]interface{}{
		"type": "widget",
		"id":   9,
		"name": "item",
	}

	obj := toAPI(data)
	require.Equal(t, "widget", obj.Type)
	require.Equal(t, "9", obj.ID)

	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "item", payload["name"])
}

