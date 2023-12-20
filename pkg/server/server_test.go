package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rancher/apiserver/pkg/builtin"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/writer"
	"github.com/stretchr/testify/require"
)

func TestServeHTMLEscaping(t *testing.T) {
	const (
		defaultJS         = "cattle.io"
		defaultCSS        = "cattle.io"
		defaultAPIVersion = "v1/apps.daemonsets.0.0"
		xss               = "<script>alert('xss')</script>"
		alphaNumeric      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		badChars          = `~!@#$%^&*()_+-=[]\{}|;':",./<>?`
	)
	xssUrl := url.URL{RawPath: xss}

	var escapedBadChars strings.Builder
	for _, r := range badChars {
		escapedBadChars.WriteString(fmt.Sprintf("&#x%X;", r))
	}

	t.Parallel()
	tests := []struct {
		name             string
		CSSURL           string
		JSURL            string
		APIUIVersion     string
		URL              string
		desiredContent   string
		undesiredContent string
	}{
		{
			name:           "base case no xss",
			CSSURL:         defaultCSS,
			JSURL:          defaultJS,
			APIUIVersion:   defaultAPIVersion,
			URL:            "https://cattle.io/v1/apps.daemonsets",
			desiredContent: "https://cattle.io/v1/apps.daemonsets",
		},
		{
			name:           "JSS alpha-numeric",
			CSSURL:         defaultCSS,
			JSURL:          alphaNumeric,
			APIUIVersion:   defaultAPIVersion,
			URL:            "https://cattle.io/v1/apps.daemonsets",
			desiredContent: alphaNumeric,
		},
		{
			name:             "JSS escaped non alpha-numeric",
			CSSURL:           defaultCSS,
			JSURL:            badChars,
			APIUIVersion:     defaultAPIVersion,
			URL:              "https://cattle.io/v1/apps.daemonsets",
			desiredContent:   escapedBadChars.String(),
			undesiredContent: badChars,
		},
		{
			name:           "CSS alpha-numeric",
			CSSURL:         alphaNumeric,
			JSURL:          defaultJS,
			APIUIVersion:   defaultAPIVersion,
			URL:            "https://cattle.io/v1/apps.daemonsets",
			desiredContent: alphaNumeric,
		},
		{
			name:             "CSS escaped non alpha-numeric",
			CSSURL:           badChars,
			JSURL:            defaultJS,
			APIUIVersion:     defaultAPIVersion,
			URL:              "https://cattle.io/v1/apps.daemonsets",
			desiredContent:   escapedBadChars.String(),
			undesiredContent: badChars,
		},
		{
			name:           "api version alpha-numeric",
			APIUIVersion:   alphaNumeric,
			URL:            "https://cattle.io/v3",
			desiredContent: alphaNumeric,
		},
		{
			name:             "api version escaped non alpha-numeric",
			APIUIVersion:     badChars,
			URL:              "https://cattle.io/v1/apps.daemonsets",
			desiredContent:   escapedBadChars.String(),
			undesiredContent: badChars,
		},
		{
			name:             "Link XSS",
			URL:              "https://cattle.io/v1/apps.daemonsets" + xss,
			undesiredContent: xss,
			desiredContent:   xssUrl.String(),
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			respStr, err := sendTestRequest(tt.URL, tt.CSSURL, tt.JSURL, tt.APIUIVersion)
			require.NoError(t, err, "failed to create server")
			require.Contains(t, respStr, tt.desiredContent, "expected content missing from server response")
			if tt.undesiredContent != "" {
				require.NotContains(t, respStr, tt.undesiredContent, "unexpected content found in server response")
			}
		})
	}
}

func sendTestRequest(url, cssURL, jssURL, apiUIVersion string) (string, error) {
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	// These header values are needed to get an HTML return document
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-agent", "Mozilla")
	srv := DefaultAPIServer()
	srv.CustomAPIUIResponseWriter(stringGetter(cssURL), stringGetter(jssURL), stringGetter(apiUIVersion))
	srv.Schemas = builtin.Schemas
	apiOp := &types.APIRequest{
		Request:  req,
		Response: resp,
		Type:     "schema",
	}
	srv.Handle(apiOp)
	return resp.Body.String(), nil
}

func stringGetter(val string) writer.StringGetter {
	return func() string { return val }
}
