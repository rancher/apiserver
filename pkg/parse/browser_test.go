package parse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBrowser(t *testing.T) {
	type testCase struct {
		description  string
		userAgent    string
		accept       string
		checkAccepts bool
		status       bool
	}
	var tests []testCase
	tests = append(tests, testCase{
		description: "Works with Mozilla UA",
		userAgent:   "Mozilla/5.0",
		accept:      "*/*",
		status:      true})
	tests = append(tests, testCase{
		description: "Fails With non-Mozilla UA",
		userAgent:   "curl/8.0",
		accept:      "*/*",
		status:      false})
	tests = append(tests, testCase{
		description:  "Fails with non-wildcard when checking is on",
		userAgent:    "Mozilla/5.0",
		accept:       "application/json",
		checkAccepts: true,
		status:       false})
	tests = append(tests, testCase{
		description:  "Accepts non-wildcard when checking is off",
		userAgent:    "Mozilla/5.0",
		accept:       "application/json",
		checkAccepts: false,
		status:       true})
	tests = append(tests, testCase{
		description:  "Accepts empty accept-header when checking is off",
		userAgent:    "Mozilla/5.0",
		checkAccepts: false,
		status:       true})
	tests = append(tests, testCase{
		description:  "Accepts empty accept-header when checking is on",
		userAgent:    "Mozilla/5.0",
		checkAccepts: true,
		status:       true})
	tests = append(tests, testCase{
		description: "Works with Mozilla UA, ignoring case",
		userAgent:   "MoZilLA/5.0",
		accept:      "*/*",
		status:      true})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("User-Agent", test.userAgent)
			req.Header.Set("Accept", test.accept)
			status := IsBrowser(req, test.checkAccepts)
			require.Equal(t, test.status, status)
		})
	}
}

func TestMatchBrowser(t *testing.T) {
	type testCase struct {
		description string
		userAgent   string
		accept      string
		status      bool
	}
	var tests []testCase
	tests = append(tests, testCase{
		description: "Works with Mozilla UA",
		userAgent:   "Mozilla/5.0",
		accept:      "*/*",
		status:      true})
	tests = append(tests, testCase{
		description: "Fails With non-Mozilla UA",
		userAgent:   "curl/8.0",
		accept:      "*/*",
		status:      false})
	tests = append(tests, testCase{
		description: "Fails with non-wildcard when checking is on",
		userAgent:   "Mozilla/5.0",
		accept:      "application/json",
		status:      false})
	tests = append(tests, testCase{
		description: "Accepts empty accept-header when checking is on",
		userAgent:   "Mozilla/5.0",
		status:      true})
	tests = append(tests, testCase{
		description: "Works with Mozilla UA, ignoring case",
		userAgent:   "MoZilLA/5.0",
		accept:      "*/*",
		status:      true})
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("User-Agent", test.userAgent)
			req.Header.Set("Accept", test.accept)
			status := MatchBrowser(req)
			require.Equal(t, test.status, status)
			status = MatchNotBrowser(req)
			require.Equal(t, !test.status, status)
		})
	}
}
