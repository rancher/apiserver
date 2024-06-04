package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/builtin"
	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/rancher/apiserver/pkg/parse"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/writer"
	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ServerSuite struct {
	suite.Suite
}

func TestServer(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServerSuite))
}

func (p *ServerSuite) TestServer_DefaultAPIServer() {
	s := DefaultAPIServer()
	assert.NotNil(p.T(), s)
	assert.NotNil(p.T(), s.Schemas)
	assert.NotNil(p.T(), s.ResponseWriters)
	assert.NotNil(p.T(), s.AccessControl)
	assert.NotNil(p.T(), s.Parser)
	assert.NotNil(p.T(), s.URLParser)
}

func (p *ServerSuite) TestServer_handle() {
	response := fakes.NewDummyWriter()
	request, _ := http.NewRequest("GET", "http://example.com", nil)

	apiRequest := new(types.APIRequest)
	apiRequest.Request = request
	apiRequest.Response = response

	type fields struct {
		ResponseWriters map[string]types.ResponseWriter
		Schemas         *types.APISchemas
		AccessControl   types.AccessControl
		Parser          parse.Parser
		URLParser       parse.URLParser
	}
	type args struct {
		apiOp  *types.APIRequest
		parser parse.Parser
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "empty Schema doesn't cause panic",
			fields: fields{
				ResponseWriters: map[string]types.ResponseWriter{},
				Schemas:         new(types.APISchemas),
				AccessControl:   &SchemaBasedAccess{},
				Parser:          parse.Parse,
				URLParser:       parse.MuxURLParser,
			},
			args: args{
				apiOp:  apiRequest,
				parser: func(apiOp *types.APIRequest, urlParser parse.URLParser) error { return nil },
			},
		},
	}
	for _, tt := range tests {
		p.Run(tt.name, func() {
			s := &Server{
				ResponseWriters: tt.fields.ResponseWriters,
				Schemas:         tt.fields.Schemas,
				AccessControl:   tt.fields.AccessControl,
				Parser:          tt.fields.Parser,
				URLParser:       tt.fields.URLParser,
			}
			s.handle(tt.args.apiOp, tt.args.parser)
		})
	}
}

func (p *ServerSuite) TestServer_handleOp() {
	ctrl := gomock.NewController(p.T())
	accessControl := fakes.NewMockAccessControl(ctrl)

	requestHandler := func(*types.APIRequest) (types.APIObject, error) { return types.APIObject{}, nil }
	expectedError := errors.New("")
	requestHandlerError := func(*types.APIRequest) (types.APIObject, error) { return types.APIObject{}, expectedError }
	requestListHandler := func(*types.APIRequest) (types.APIObjectList, error) { return types.APIObjectList{}, nil }
	handler := fakes.DummyHandler{}

	type fields struct {
		Action        string
		Link          string
		Method        string
		Name          string
		Schema        *types.APISchema
		AccessControl types.AccessControl
		Headers       map[string]string
	}
	type results struct {
		Code int
		Data interface{}
		Err  error
	}
	tests := []struct {
		name    string
		fields  fields
		results results
	}{
		{
			name: "Bad CSRF header",
			fields: fields{
				Headers: map[string]string{
					"User-Agent": "mozilla",
					"Cookie":     "CSRF=test",
				},
			},
			results: results{
				Code: 0,
				Data: nil,
				Err:  apierror.NewAPIError(validation.InvalidCSRFToken, "Invalid CSRF token"),
			},
		},
		{
			name:   "Request with nil schema",
			fields: fields{},
			results: results{
				Code: http.StatusNotFound,
				Data: nil,
				Err:  nil,
			},
		},
		{
			name: "Empty Schema",
			fields: fields{
				Schema: &types.APISchema{},
			},
			results: results{
				Code: http.StatusNotFound,
				Data: nil,
				Err:  nil,
			},
		},
		{
			name: "GET List Request",
			fields: fields{
				Schema: &types.APISchema{
					ListHandler: requestListHandler,
				},
				Method: http.MethodGet,
			},
			results: results{
				Code: http.StatusOK,
				Data: types.APIObjectList{},
				Err:  nil,
			},
		},
		{
			name: "GET Request",
			fields: fields{
				Schema: &types.APISchema{
					ByIDHandler: requestHandler,
				},
				Method: http.MethodGet,
				Name:   ".",
			},
			results: results{
				Code: http.StatusOK,
				Data: types.APIObject{},
				Err:  nil,
			},
		},
		{
			name: "PATCH Request",
			fields: fields{
				Schema: &types.APISchema{
					UpdateHandler: requestHandler,
				},
				Method: http.MethodPatch,
			},
			results: results{
				Code: http.StatusOK,
				Data: types.APIObject{},
				Err:  nil,
			},
		},
		{
			name: "POST Request",
			fields: fields{
				Schema: &types.APISchema{
					CreateHandler: requestHandler,
				},
				Method: http.MethodPost,
			},
			results: results{
				Code: http.StatusCreated,
				Data: types.APIObject{},
				Err:  nil,
			},
		},
		{
			name: "DELETE Request",
			fields: fields{
				Schema: &types.APISchema{
					DeleteHandler: requestHandler,
				},
				Method: http.MethodDelete,
			},
			results: results{
				Code: http.StatusOK,
				Data: types.APIObject{},
				Err:  nil,
			},
		},
		{
			name: "Validated POST Request",
			fields: fields{
				Schema: &types.APISchema{
					Schema: &schemas.Schema{
						CollectionActions: map[string]schemas.Action{"POST": schemas.Action{}},
					},
					ActionHandlers: map[string]http.Handler{"POST": &handler},
				},
				Action:        "POST",
				Link:          "",
				Method:        http.MethodPost,
				Name:          "",
				AccessControl: accessControl,
			},
			results: results{
				Code: http.StatusOK,
				Data: nil,
				Err:  validation.ErrComplete,
			},
		},
		{
			name: "Validated Named POST Request",
			fields: fields{
				Schema: &types.APISchema{
					Schema: &schemas.Schema{
						CollectionActions: map[string]schemas.Action{"POST": schemas.Action{}},
						ResourceActions:   map[string]schemas.Action{"POST": schemas.Action{}},
					},
					ActionHandlers: map[string]http.Handler{"POST": &handler},
					ByIDHandler:    requestHandlerError,
				},
				Action:        "POST",
				Link:          "",
				Method:        http.MethodPost,
				Name:          "TEST",
				AccessControl: accessControl,
			},
			results: results{
				Code: http.StatusOK,
				Data: types.APIObject{},
				Err:  expectedError,
			},
		},
	}
	for _, tt := range tests {
		p.Run(tt.name, func() {
			s := &Server{}
			apiRequest := &types.APIRequest{
				Action:        tt.fields.Action,
				Link:          tt.fields.Link,
				Method:        tt.fields.Method,
				Name:          tt.fields.Name,
				Schema:        tt.fields.Schema,
				AccessControl: tt.fields.AccessControl,
			}

			req, _ := http.NewRequest("", "", nil)
			apiRequest.Request = req
			for k, v := range tt.fields.Headers {
				apiRequest.Request.Header.Add(k, v)
			}

			if apiRequest.AccessControl != nil {
				ac := apiRequest.AccessControl.(*fakes.MockAccessControl)
				ac.EXPECT().CanAction(apiRequest, apiRequest.Schema, apiRequest.Action).Return(nil).AnyTimes()
			}

			c, d, e := s.handleOp(apiRequest)
			assert.Equal(p.T(), tt.results.Code, c)
			assert.Equal(p.T(), tt.results.Data, d)
			assert.Equal(p.T(), tt.results.Err, e)
		})
	}
}

func (p *ServerSuite) TestServer_handleAction() {
	ctrl := gomock.NewController(p.T())
	accessControl := fakes.NewMockAccessControl(ctrl)

	handler := fakes.DummyHandler{}

	schema := new(types.APISchema)
	schema.ActionHandlers = map[string]http.Handler{}
	schema.ActionHandlers[""] = &handler

	apiRequest := new(types.APIRequest)
	apiRequest.AccessControl = accessControl

	// If CanAction returns an error, get that back
	expected_err := errors.New("")
	accessControl.EXPECT().CanAction(apiRequest, nil, "").Return(expected_err)
	err := handleAction(apiRequest)
	assert.Equal(p.T(), err, expected_err)

	// If schema has the right ActionHandler return ErrComplete
	accessControl.EXPECT().CanAction(apiRequest, schema, "").Return(nil)
	apiRequest.Schema = schema
	err = handleAction(apiRequest)
	assert.Equal(p.T(), err, validation.ErrComplete)

	// If schema does not have the right ActionHandler, we get nil
	accessControl.EXPECT().CanAction(apiRequest, schema, "GET").Return(nil)
	apiRequest.Action = "GET"
	err = handleAction(apiRequest)
	assert.Nil(p.T(), err)
}

func (p *ServerSuite) TestServer_CustomAPIUIResponseWriter() {
	d := &writer.GzipWriter{
		ResponseWriter: &writer.HTMLResponseWriter{
			CSSURL:       nil,
			JSURL:        nil,
			APIUIVersion: nil,
		},
	}
	f := func() string { return "" }

	s := &Server{
		ResponseWriters: map[string]types.ResponseWriter{},
	}
	w := d.ResponseWriter.(*writer.HTMLResponseWriter)

	// If there is not an html entry, do not update
	s.CustomAPIUIResponseWriter(f, f, f)
	assert.Nil(p.T(), w.CSSURL)
	assert.Nil(p.T(), w.JSURL)
	assert.Nil(p.T(), w.APIUIVersion)

	s.ResponseWriters["html"] = d

	// Now we should update
	s.CustomAPIUIResponseWriter(f, f, f)
	assert.NotNil(p.T(), w.CSSURL)
	assert.NotNil(p.T(), w.JSURL)
	assert.NotNil(p.T(), w.APIUIVersion)
}

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
