package parse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandardURLParserUsesPathValuesForActionAndLink(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos/one?action=queryAction&link=queryLink", nil)
	req.SetPathValue("type", "foos")
	req.SetPathValue("name", "one")
	req.SetPathValue("action", "pathAction")
	req.SetPathValue("link", "pathLink")

	parsed, err := StandardURLParser(httptest.NewRecorder(), req, nil)
	require.NoError(t, err)
	require.Equal(t, "pathAction", parsed.Action)
	require.Equal(t, "pathLink", parsed.Link)
	require.Equal(t, "foos", parsed.Type)
	require.Equal(t, "one", parsed.Name)
}

func TestStandardURLParserFallsBackToQueryWhenPathValuesMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos/one?action=queryAction&link=queryLink", nil)
	req.SetPathValue("type", "foos")
	req.SetPathValue("name", "one")

	parsed, err := StandardURLParser(httptest.NewRecorder(), req, nil)
	require.NoError(t, err)
	require.Equal(t, "queryAction", parsed.Action)
	require.Equal(t, "queryLink", parsed.Link)
}

