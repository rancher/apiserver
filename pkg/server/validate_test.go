package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func TestCheckCSRFMethodOverrideUsesAPIMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos?_method=POST", nil)
	req.Header.Set("User-Agent", "Mozilla")
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "token-1"})

	apiReq := &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
		Method:   http.MethodPost,
	}

	err := CheckCSRF(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidCSRFToken, apiErr.Code)
}

func TestCheckCSRFSkipsTokenValidationForGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos", nil)
	req.Header.Set("User-Agent", "Mozilla")
	req.Header.Set(csrfHeader, "wrong-header-token")
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "cookie-token"})

	apiReq := &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
		Method:   http.MethodGet,
	}

	err := CheckCSRF(apiReq)
	require.NoError(t, err)
}

func TestCheckCSRFIssuesCookieWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos", nil)
	req.Header.Set("User-Agent", "Mozilla")
	rw := httptest.NewRecorder()

	apiReq := &types.APIRequest{
		Request:  req,
		Response: rw,
		Method:   http.MethodGet,
	}

	err := CheckCSRF(apiReq)
	require.NoError(t, err)

	result := rw.Result()
	cookies := result.Cookies()
	require.NotEmpty(t, cookies)

	var csrf *http.Cookie
	for _, c := range cookies {
		if c.Name == csrfCookie {
			csrf = c
			break
		}
	}
	require.NotNil(t, csrf)
	require.NotEmpty(t, csrf.Value)
	require.Equal(t, "/", csrf.Path)
	require.True(t, csrf.Secure)
}

