package subscribe

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v2/pkg/schemas"
	"github.com/stretchr/testify/assert"
)

func Test_stream(t *testing.T) {
	tests := []struct {
		name           string
		sub            Subscribe
		hasAccess      bool
		wantStartEvent types.APIEvent
		wantError      bool
	}{
		{
			name: "stream all",
			sub: Subscribe{
				ResourceType: "watchable-resource",
			},
			hasAccess: true,
			wantStartEvent: types.APIEvent{
				Name:         "resource.start",
				ResourceType: "watchable-resource",
			},
		},
		{
			name: "stream by namespace",
			sub: Subscribe{
				ResourceType: "watchable-resource",
				Namespace:    "test-ns",
			},
			hasAccess: true,
			wantStartEvent: types.APIEvent{
				Name:         "resource.start",
				ResourceType: "watchable-resource",
				Namespace:    "test-ns",
			},
		},
		{
			name: "stream by selector",
			sub: Subscribe{
				ResourceType: "watchable-resource",
				Selector:     "foo=bar",
			},
			hasAccess: true,
			wantStartEvent: types.APIEvent{
				Name:         "resource.start",
				ResourceType: "watchable-resource",
				Selector:     "foo=bar",
			},
		},
		{
			name: "stream by id",
			sub: Subscribe{
				ResourceType: "watchable-resource",
				ID:           "test-resource",
			},
			hasAccess: true,
			wantStartEvent: types.APIEvent{
				Name:         "resource.start",
				ResourceType: "watchable-resource",
				ID:           "test-resource",
			},
		},
		{
			name: "missing schema error",
			sub: Subscribe{
				ResourceType: "notaresource",
			},
			hasAccess: true,
			wantError: true,
		},
		{
			name: "unsupported schema error",
			sub: Subscribe{
				ResourceType: "listonly-resource",
			},
			hasAccess: true,
			wantError: true,
		},
		{
			name: "forbidden schema error",
			sub: Subscribe{
				ResourceType: "watchable-resource",
			},
			hasAccess: false,
			wantError: true,
		},
	}
	ws := WatchSession{
		apiOp: &types.APIRequest{
			Name: "test",
			Schemas: &types.APISchemas{
				Schemas: map[string]*types.APISchema{
					"watchable-resource": {
						Schema: &schemas.Schema{
							ID: "watchable-resource",
						},
						Store: &mockStore{},
					},
					"listonly-resource": {
						Schema: &schemas.Schema{
							ID: "listonly-resource",
						},
					},
				},
			},
			Request: &http.Request{},
		},
		getter: DefaultGetter,
	}
	for _, test := range tests {
		ws.apiOp.AccessControl = &mockAC{hasAccess: test.hasAccess}
		t.Run(test.name, func(t *testing.T) {
			result := make(chan types.APIEvent, 1)
			err := ws.stream(context.TODO(), test.sub, result)
			if test.wantError {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			var gotEvent types.APIEvent
			select {
			case gotEvent = <-result:
			case <-time.After(10 * time.Millisecond):
				assert.FailNow(t, "failed to receive startup message from websocket")
			}
			assert.Equal(t, test.wantStartEvent, gotEvent)
		})
	}
}

type mockStore struct{}

func (m *mockStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	panic("not implemented")
}

func (m *mockStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	panic("not implemented")
}

func (m *mockStore) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	panic("not implemented")
}

func (m *mockStore) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	panic("not implemented")
}

func (m *mockStore) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	panic("not implemented")
}

func (m *mockStore) Watch(apiOp *types.APIRequest, schema *types.APISchema, w types.WatchRequest) (chan types.APIEvent, error) {
	c := make(chan types.APIEvent)
	go func() {
		c <- types.APIEvent{}
		close(c)
	}()
	return c, nil
}

type mockAC struct {
	hasAccess bool
}

func (m *mockAC) CanAction(apiOp *types.APIRequest, schema *types.APISchema, name string) error {
	panic("not implemented")
}

func (m *mockAC) CanCreate(apiOp *types.APIRequest, schema *types.APISchema) error {
	panic("not implemented")
}

func (m *mockAC) CanList(apiOp *types.APIRequest, schema *types.APISchema) error {
	panic("not implemented")
}

func (m *mockAC) CanGet(apiOp *types.APIRequest, schema *types.APISchema) error {
	panic("not implemented")
}

func (m *mockAC) CanUpdate(apiOp *types.APIRequest, obj types.APIObject, schema *types.APISchema) error {
	panic("not implemented")
}

func (m *mockAC) CanDelete(apiOp *types.APIRequest, obj types.APIObject, schema *types.APISchema) error {
	panic("not implemented")
}

func (m *mockAC) CanWatch(apiOp *types.APIRequest, schema *types.APISchema) error {
	if m.hasAccess {
		return nil
	}
	return fmt.Errorf("forbidden")
}

func (m *mockAC) CanDo(apiOp *types.APIRequest, resource, verb, namespace, name string) error {
	panic("not implemented")
}
