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

func newByIDRequest(name, link string, ac types.AccessControl, store types.Store, linkHandlers map[string]http.Handler) *types.APIRequest {
	req := httptest.NewRequest(http.MethodGet, "/v1/blocks/"+name, nil)
	return &types.APIRequest{
		Request:  req,
		Response: httptest.NewRecorder(),
		Name:     name,
		Link:     link,
		Schema: &types.APISchema{
			Store:        store,
			LinkHandlers: linkHandlers,
		},
		AccessControl: ac,
	}
}

// ByIDHandler tests

func TestByIDHandler_AccessDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newByIDRequest("block", "", ac, nil, nil)

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).
		Return(apierror.NewAPIError(validation.PermissionDenied, "can not get bricks"))

	_, err := ByIDHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func TestByIDHandler_NoStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newByIDRequest("two", "", ac, nil, nil)
	ac.EXPECT().CanGet(gomock.Any(), gomock.Any())

	_, err := ByIDHandler(apiReq)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no store found")
}

func TestByIDHandler_StoreLookupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newByIDRequest("goopy", "", ac, store, nil)
	storeErr := errors.New("can't find goopy")

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().ByID(gomock.Any(), gomock.Any(), "goopy").Return(types.APIObject{}, storeErr)

	_, err := ByIDHandler(apiReq)
	require.ErrorIs(t, err, storeErr)
}

func TestByIDHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newByIDRequest("one", "", ac, store, nil)

	expected := types.APIObject{ID: "one", Type: "blocks", Object: map[string]interface{}{"name": "one"}}

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().ByID(gomock.Any(), gomock.Any(), "one").Return(expected, nil)

	obj, err := ByIDHandler(apiReq)
	require.NoError(t, err)
	require.Equal(t, "one", obj.ID)
}

func TestByIDHandler_LinkHandlerInvokedWhenLinkMatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)

	invoked := false
	linkHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		invoked = true
	})
	linkHandlers := map[string]http.Handler{"log": linkHandler}
	apiReq := newByIDRequest("two", "log", ac, store, linkHandlers)

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().ByID(gomock.Any(), gomock.Any(), "two").Return(
		types.APIObject{ID: "two", Type: "blocks"}, nil,
	)

	obj, err := ByIDHandler(apiReq)
	require.ErrorIs(t, err, validation.ErrComplete)
	require.Equal(t, types.APIObject{}, obj)
	require.True(t, invoked)
}

func TestByIDHandler_UnknownLinkReturnsNormalResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)

	linkHandlers := map[string]http.Handler{"log": &fakes.DummyHandler{}}
	apiReq := newByIDRequest("three", "audit", ac, store, linkHandlers)

	expected := types.APIObject{ID: "three", Type: "blocks"}

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().ByID(gomock.Any(), gomock.Any(), "three").Return(expected, nil)

	obj, err := ByIDHandler(apiReq)
	require.NoError(t, err)
	require.Equal(t, "three", obj.ID)
}

// ListHandler tests

func newListRequest(name string, ac types.AccessControl, store types.Store) *types.APIRequest {
	req := httptest.NewRequest(http.MethodGet, "/v1/bricks", nil)
	return &types.APIRequest{
		Request: req,
		Name:    name,
		Schema: &types.APISchema{
			Store: store,
		},
		AccessControl: ac,
	}
}

func TestListHandler_AccessDeniedWithoutName(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newListRequest("", ac, nil)

	ac.EXPECT().CanList(gomock.Any(), gomock.Any()).
		Return(apierror.NewAPIError(validation.PermissionDenied, "can not list bricks"))

	_, err := ListHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func TestListHandler_AccessDeniedWithName(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newListRequest("blocked", ac, nil)

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).
		Return(apierror.NewAPIError(validation.PermissionDenied, "can not get bricks"))

	_, err := ListHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.PermissionDenied, apiErr.Code)
}

func TestListHandler_NoStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newListRequest("", ac, nil)

	ac.EXPECT().CanList(gomock.Any(), gomock.Any()).Return(nil)

	_, err := ListHandler(apiReq)
	require.Error(t, err)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.NotFound, apiErr.Code)
}

func TestListHandler_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newListRequest("", ac, store)
	storeErr := errors.New("db timeout")

	ac.EXPECT().CanList(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().List(gomock.Any(), gomock.Any()).Return(types.APIObjectList{}, storeErr)

	_, err := ListHandler(apiReq)
	require.ErrorIs(t, err, storeErr)
}

func TestListHandler_SuccessWithNoName(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newListRequest("", ac, store)

	expected := types.APIObjectList{Objects: []types.APIObject{{ID: "a"}, {ID: "b"}}}

	ac.EXPECT().CanList(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().List(gomock.Any(), gomock.Any()).Return(expected, nil)

	list, err := ListHandler(apiReq)
	require.NoError(t, err)
	require.Len(t, list.Objects, 2)
}

func TestListHandler_SuccessWithName(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newListRequest("specific", ac, store)

	ac.EXPECT().CanGet(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().List(gomock.Any(), gomock.Any()).Return(types.APIObjectList{}, nil)

	list, err := ListHandler(apiReq)
	require.NoError(t, err)
	require.Len(t, list.Objects, 0)
}
