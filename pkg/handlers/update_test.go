package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/require"
)

func newUpdateAPIRequest(method, path, body, name string, ac types.AccessControl, store types.Store) *types.APIRequest {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	return &types.APIRequest{
		Request: req,
		Name:    name,
		Schema: &types.APISchema{
			Store: store,
		},
		AccessControl: ac,
	}
}

func TestUpdateHandlerNoAccessFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPut, "/v1/football/seahawks", "", "", ac, store)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("lions"))

	_, err := UpdateHandler(apiReq)
	require.Error(t, err)
	require.Equal(t, "lions", err.Error())
}

func TestUpdateHandlerPatchBadJSONStillStores(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPatch, "/v1/ringette/nova", `{invalid-json`, "blanchette", ac, store)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), "blanchette").DoAndReturn(
		func(_ *types.APIRequest, _ *types.APISchema, data types.APIObject, _ string) (types.APIObject, error) {
			require.Equal(t, types.APIObject{}, data)
			return types.APIObject{ID: "nova", Type: "ringette", Object: map[string]interface{}{"name": "patched"}}, nil
		},
	)

	obj, err := UpdateHandler(apiReq)
	require.NoError(t, err)
	require.Equal(t, "nova", obj.ID)
}

func TestUpdateHandlerPutParseFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPut, "/v1/hockey/wings", `{"nameonly"}`, "two", ac, store)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, err := UpdateHandler(apiReq)
	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
}

func TestUpdateHandlerPutParsesBodyAndStoresIt(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPut, "/v1/baseball/orioles", `{"name":"tigers"}`, "jays", ac, store)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), "jays").DoAndReturn(
		func(_ *types.APIRequest, _ *types.APISchema, data types.APIObject, _ string) (types.APIObject, error) {
			payload, ok := data.Object.(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "tigers", payload["name"])
			return types.APIObject{ID: "tigers", Type: "orioles", Object: payload}, nil
		},
	)
	obj, err := UpdateHandler(apiReq)
	require.NoError(t, err)
	require.Equal(t, "tigers", obj.ID)
	require.NoError(t, err)
}

func TestUpdateHandlerWithoutStoreReturnsNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPut, "/v1/derby/bombers", `{"name":"hutchinson"}`, "three", ac, nil)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, err := UpdateHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.NotFound, apiErr.Code)
}

func TestUpdateHandlerStoreFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newUpdateAPIRequest(http.MethodPut, "/v1/baseball/pirates", `{"name":"stargell"}`, "catfish", ac, store)

	ac.EXPECT().CanUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), "catfish").Return(types.APIObject{}, errors.New("rollie"))
	_, err := UpdateHandler(apiReq)
	require.Error(t, err)
	require.Equal(t, "rollie", err.Error())
}
