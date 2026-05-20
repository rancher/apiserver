package handlers

import (
	"bytes"
	"errors"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/require"
)

func newAPIRequest(ac types.AccessControl, store types.Store, body string) *types.APIRequest {
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	return &types.APIRequest{
		Request:       req,
		Schema:        &types.APISchema{Store: store},
		AccessControl: ac,
	}
}

func TestCreateHandler_CantCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newAPIRequest(ac, nil, `{"name":"test"}`)

	ac.EXPECT().CanCreate(gomock.Any(), gomock.Any()).Return(errors.New("nope"))

	_, err := CreateHandler(apiReq)
	require.Error(t, err)
	require.Equal(t, "nope", err.Error())
}

func TestCreateHandler_CantParse(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newAPIRequest(ac, store, `{unquoted-and-cut-off`)

	ac.EXPECT().CanCreate(gomock.Any(), gomock.Any())

	_, err := CreateHandler(apiReq)
	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	require.Equal(t, validation.InvalidBodyContent, apiErr.Code)
}

func TestCreateHandler_NoStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	apiReq := newAPIRequest(ac, nil, `{"problem": "no store here"}`)
	ac.EXPECT().CanCreate(gomock.Any(), gomock.Any())

	_, err := CreateHandler(apiReq)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no store found")
}

func TestCreateHandler_CreateStoreFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newAPIRequest(ac, store, `{"problem": "storing fails"}`)
	ac.EXPECT().CanCreate(gomock.Any(), gomock.Any())
	store.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(types.APIObject{}, errors.New("create-store fails"))

	_, err := CreateHandler(apiReq)

	require.Error(t, err)
	require.Contains(t, err.Error(), "create-store fails")
}

func TestCreateHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ac := fakes.NewMockAccessControl(ctrl)
	store := fakes.NewMockStore(ctrl)
	apiReq := newAPIRequest(ac, store, `{"name":"test"}`)

	ac.EXPECT().CanCreate(gomock.Any(), gomock.Any())
	store.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(types.APIObject{Object: map[string]interface{}{"name": "test"}}, nil)

	obj, err := CreateHandler(apiReq)
	require.NoError(t, err)

	m, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "test", m["name"])
}
