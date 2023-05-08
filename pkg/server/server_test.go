package server

import (
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/rancher/apiserver/pkg/parse"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/suite"
)

type ServerSuite struct {
	suite.Suite
}

func TestServer(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServerSuite))
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
