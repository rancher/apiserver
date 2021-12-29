package handlers

import (
	"fmt"
	"strconv"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/metrics"
	"github.com/rancher/apiserver/pkg/types"
)

func MetricsHandler(successCode string, next func(apiRequest *types.APIRequest) (types.APIObject, error)) func(apiRequest *types.APIRequest) (types.APIObject, error) {
	return func(request *types.APIRequest) (types.APIObject, error) {
		obj, err := next(request)
		if err != nil {
			if apiError, ok := err.(*apierror.APIError); ok {

				metrics.IncTotalResponses(request.Schema.ID, request.Method, strconv.Itoa(apiError.Code.Status), fmt.Sprintf("%s:%s", request.Namespace, request.Name))
			}
			return types.APIObject{}, err
		}

		metrics.IncTotalResponses(request.Schema.ID, request.Method, successCode, fmt.Sprintf("%s:%s", request.Namespace, request.Name))
		return obj, err
	}
}

func MetricsListHandler(successCode string, next func(apiRequest *types.APIRequest) (types.APIObjectList, error)) func(apiRequest *types.APIRequest) (types.APIObjectList, error) {
	return func(request *types.APIRequest) (types.APIObjectList, error) {
		var id string
		if request.Name != "" {
			id = fmt.Sprintf("%s:%s", request.Namespace, request.Name)
		} else {
			id = request.Namespace
		}

		objList, err := next(request)
		if err != nil {
			if apiError, ok := err.(*apierror.APIError); ok {
				metrics.IncTotalResponses(request.Schema.ID, request.Method, strconv.Itoa(apiError.Code.Status), id)
			}
			return types.APIObjectList{}, err
		}

		metrics.IncTotalResponses(request.Schema.ID, request.Method, successCode, id)
		return objList, err
	}
}