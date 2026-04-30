package handlers

import (
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

func newDeleteAPIRequest(name string, ac types.AccessControl, store types.Store) *types.APIRequest {
	req := httptest.NewRequest(http.MethodDelete, "/v1/resourceToDelete/"+name, nil)
	return &types.APIRequest{
		Request: req,
		Name:    name,
		Schema: &types.APISchema{
			Store: store,
		},
		AccessControl: ac,
	}
}

func TestDeleteHandler_CantDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newDeleteAPIRequest("forbidden", ac, nil)

	ac.EXPECT().CanDelete(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(apierror.NewAPIError(validation.PermissionDenied, "can not delete resourceToDelete"))

	_, err := DeleteHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func TestDeleteHandler_NoStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newDeleteAPIRequest("no-store-here", ac, nil)
	ac.EXPECT().CanDelete(gomock.Any(), gomock.Any(), gomock.Any())

	_, err := DeleteHandler(apiReq)

	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.NotFound, apiErr.Code)
	require.Contains(t, err.Error(), "no store found")
}

func TestDeleteHandler_StoreDeleteFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newDeleteAPIRequest("gone", ac, store)
	storeErr := errors.New("store exploded")

	ac.EXPECT().CanDelete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().Delete(gomock.Any(), gomock.Any(), "gone").Return(types.APIObject{}, storeErr)

	_, err := DeleteHandler(apiReq)
	require.ErrorIs(t, err, storeErr)
}

func TestDeleteHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newDeleteAPIRequest("bricks", ac, store)

	deleted := types.APIObject{ID: "bricks", Type: "resourceToDelete", Object: map[string]interface{}{"name": "bricks"}}

	ac.EXPECT().CanDelete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().Delete(gomock.Any(), gomock.Any(), "bricks").Return(deleted, nil)

	obj, err := DeleteHandler(apiReq)
	require.NoError(t, err)
	require.Equal(t, "bricks", obj.ID)
}
