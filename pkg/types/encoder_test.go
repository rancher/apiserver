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
