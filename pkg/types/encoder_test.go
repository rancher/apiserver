package types_test

import (
	"bytes"
	"testing"

	"github.com/rancher/apiserver/pkg/types"
)

func TestJSONLinesEncoder(t *testing.T) {
	collection := types.Collection{
		Links:        map[string]string{},
		Actions:      map[string]string{},
		ResourceType: "Test",
	}

	data := []*types.RawResource{
		{},
		{},
		{},
	}

	type args struct {
		v interface{}
	}
	tests := []struct {
		name       string
		args       args
		wantWriter string
		wantErr    bool
	}{
		{
			name:       "empty collection list",
			args:       args{&types.GenericCollection{collection, []*types.RawResource{}, []types.SummaryEntry{}}},
			wantWriter: "{\"links\":{},\"actions\":{},\"resourceType\":\"Test\"}\n\n",
		},
		{
			name:       "valid collection list",
			args:       args{&types.GenericCollection{collection, data, []types.SummaryEntry{}}},
			wantWriter: "{\"links\":{},\"actions\":{},\"resourceType\":\"Test\"}\n{\"links\":null}\n{\"links\":null}\n{\"links\":null}\n\n",
		},
		{
			name: "valid collection list with summary",
			args: args{&types.GenericCollection{Collection: collection, Data: data, Summary: []types.SummaryEntry{
				types.SummaryEntry{
					Property: "field01",
					Counts: map[string]types.SummaryWithBreakdown{
						"walrus": {Total: 3, Namespace: map[string]int{"zoo": 1, "park": 2}},
						"cat":    {Total: 4, Namespace: map[string]int{"house": 3, "nursing home": 1}},
					},
				},
				types.SummaryEntry{
					Property: "field02",
					Counts: map[string]types.SummaryWithBreakdown{
						"walrus": {Total: 5, Namespace: map[string]int{"zoo": 3, "park": 2}},
						"cat":    {Total: 2, Namespace: map[string]int{"house": 1, "nursing home": 1}},
					},
				}}}},
			wantWriter: `{"links":{},"actions":{},"resourceType":"Test"}
{"links":null}
{"links":null}
{"links":null}
{"property":"field01","counts":{"cat":{"total":4,"namespace":{"house":3,"nursing home":1}},"walrus":{"total":3,"namespace":{"park":2,"zoo":1}}}}
{"property":"field02","counts":{"cat":{"total":2,"namespace":{"house":1,"nursing home":1}},"walrus":{"total":5,"namespace":{"park":2,"zoo":3}}}}

`,
		},
		{
			name:       "arbitrary type",
			args:       args{"foobarbaz"},
			wantWriter: "\"foobarbaz\"\n\n",
		},
		{
			name:    "invalid type",
			args:    args{func() {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			err := types.JSONLinesEncoder(writer, tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONLinesEncoder() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotWriter := writer.String(); gotWriter != tt.wantWriter {
				t.Errorf("JSONLinesEncoder() gotWriter = %v, want %v", gotWriter, tt.wantWriter)
			}
		})
	}
}

func TestYAMLEncoder(t *testing.T) {
	tests := []struct {
		name    string
		v       interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple object",
			v:    map[string]string{"hello": "world"},
			want: "hello: world\n",
		},
		{
			name: "nested object",
			v:    map[string]interface{}{"meta": map[string]string{"name": "foo"}},
			want: "meta:\n  name: foo\n",
		},
		{
			name:    "unmarshalable type",
			v:       func() {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			err := types.YAMLEncoder(writer, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLEncoder() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotWriter := writer.String(); !tt.wantErr && gotWriter != tt.want {
				t.Errorf("YAMLEncoder() gotWriter = %q, want %q", gotWriter, tt.want)
			}
		})
	}
}
