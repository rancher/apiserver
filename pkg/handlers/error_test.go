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

func newErrorAPIRequest(rw http.ResponseWriter, responseWriter types.ResponseWriter) *types.APIRequest {
	req := httptest.NewRequest(http.MethodGet, "/v1/foos/one", nil)
	return &types.APIRequest{
		Request:        req,
		Response:       rw,
		ResponseWriter: responseWriter,
	}
}

func TestErrorHandler_IgnoresErrComplete(t *testing.T) {
	rw := httptest.NewRecorder()
	apiReq := newErrorAPIRequest(rw, nil)

	ErrorHandler(apiReq, validation.ErrComplete)

	require.Equal(t, http.StatusOK, rw.Code)
}

func TestErrorHandler_ConvertsValidationErrorCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := fakes.NewMockResponseWriter(ctrl)
	rw := httptest.NewRecorder()
	apiReq := newErrorAPIRequest(rw, writer)

	writer.EXPECT().Write(gomock.Any(), validation.NotFound.Status, gomock.Any()).Do(
		func(_ *types.APIRequest, _ int, obj types.APIObject) {
			require.Equal(t, "error", obj.Type)
			payload, ok := obj.Object.(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, validation.NotFound.Status, payload["status"])
			require.Equal(t, validation.NotFound.Code, payload["code"])
			require.Equal(t, "", payload["message"])
		},
	)

	ErrorHandler(apiReq, validation.NotFound)
}

func TestErrorHandler_UsesAPIErrorPayloadIncludingFieldName(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := fakes.NewMockResponseWriter(ctrl)
	rw := httptest.NewRecorder()
	apiReq := newErrorAPIRequest(rw, writer)

	err, ok := apierror.NewAPIError(validation.InvalidFormat, "bad field").(*apierror.APIError)
	require.True(t, ok)
	err.FieldName = "metadata.name"

	writer.EXPECT().Write(gomock.Any(), validation.InvalidFormat.Status, gomock.Any()).Do(
		func(_ *types.APIRequest, _ int, obj types.APIObject) {
			payload, ok := obj.Object.(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "metadata.name", payload["fieldName"])
			require.Equal(t, "bad field", payload["message"])
			require.Equal(t, validation.InvalidFormat.Status, payload["status"])
			require.Equal(t, validation.InvalidFormat.Code, payload["code"])
		},
	)

	ErrorHandler(apiReq, err)
}

func TestErrorHandler_WrapsUnknownErrorAsServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := fakes.NewMockResponseWriter(ctrl)
	rw := httptest.NewRecorder()
	apiReq := newErrorAPIRequest(rw, writer)

	writer.EXPECT().Write(gomock.Any(), validation.ServerError.Status, gomock.Any()).Do(
		func(_ *types.APIRequest, _ int, obj types.APIObject) {
			payload, ok := obj.Object.(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, validation.ServerError.Code, payload["code"])
			require.Equal(t, validation.ServerError.Status, payload["status"])
			require.Equal(t, "boom", payload["message"])
		},
	)

	ErrorHandler(apiReq, errors.New("boom"))
}

func TestErrorHandler_WritesNoContentStatusWithoutResponseWriter(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := fakes.NewMockResponseWriter(ctrl)
	rw := httptest.NewRecorder()
	apiReq := newErrorAPIRequest(rw, writer)

	err := &apierror.APIError{
		Code: validation.ErrorCode{
			Code:   "NoContent",
			Status: http.StatusNoContent,
		},
		Message: "",
	}

	ErrorHandler(apiReq, err)

	require.Equal(t, http.StatusNoContent, rw.Code)
}

func TestToError_IncludesFieldNameWhenSet(t *testing.T) {
	err, ok := apierror.NewAPIError(validation.InvalidBodyContent, "invalid body").(*apierror.APIError)
	require.True(t, ok)
	err.FieldName = "spec.replicas"

	obj := toError(err)
	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)

	require.Equal(t, "error", obj.Type)
	require.Equal(t, "spec.replicas", payload["fieldName"])
	require.Equal(t, validation.InvalidBodyContent.Status, payload["status"])
}

func TestToError_OmitsFieldNameWhenEmpty(t *testing.T) {
	err, ok := apierror.NewAPIError(validation.PermissionDenied, "denied").(*apierror.APIError)
	require.True(t, ok)

	obj := toError(err)
	payload, ok := obj.Object.(map[string]interface{})
	require.True(t, ok)

	_, found := payload["fieldName"]
	require.False(t, found)
	require.Equal(t, "denied", payload["message"])
}
