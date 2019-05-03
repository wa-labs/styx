package kitty

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

type errBodyDecoding interface {
	error
	IsErrBodyDecoding()
}

type errBodyDecodingBehavior struct{}

func (e errBodyDecodingBehavior) IsErrBodyDecoding() {}

// WithErrBodyDecoding adds a errBodyDecodingBehavior to the given error.
func WithErrBodyDecoding(err error) error {
	return struct {
		error
		errBodyDecodingBehavior
	}{
		err,
		errBodyDecodingBehavior{},
	}
}

func isErrBodyDecoding(err error) bool {
	_, ok := err.(errBodyDecoding)
	return ok
}

type errQueryParam interface {
	error
	IsErrQueryParam()
	QueryParamKey() string
}

type errQueryParamBehavior struct {
	key string
}

func (e errQueryParamBehavior) IsErrQueryParam()      {}
func (e errQueryParamBehavior) QueryParamKey() string { return e.key }

// WithErrQueryParam adds a errQueryParamBehavior to the given error.
func WithErrQueryParam(err error, key string) error {
	return struct {
		error
		errQueryParamBehavior
	}{
		err,
		errQueryParamBehavior{
			key: key,
		},
	}
}

func isErrQueryParam(err error) (string, bool) {
	if e, ok := err.(errQueryParam); ok {
		return e.QueryParamKey(), true
	}
	return "", false
}

// TransportErrorEncoder offers a standardized way of handling transport errors.
// It responds with the appropriate APIError and traces it.
func TransportErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	var apiError APIError
	// switch e1 := err.(type) {
	// case httptransport.Error:
	// 	err = e1.Err
	// 	switch e1.Domain {
	// case httptransport.DomainDecode:
	// 	if isErrBodyDecoding(err) {
	// 		apiError = APIBodyDecoding
	// 	} else if key, ok := isErrQueryParam(err); ok {
	// 		apiError = APIQueryParam
	// 		apiError.Params["key"] = key
	// 	} else {
	// 		apiError = APIInternal
	// 		TraceError(ctx, err)
	// 	}
	// case httptransport.DomainDo:
	// 	apiError = APIUnavailable
	// 	TraceError(ctx, err)
	// 	default:
	// 		apiError = APIInternal
	// 		TraceError(ctx, err)
	// 	}
	// default:
	// 	apiError = APIInternal
	// 	TraceError(ctx, err)
	// }

	apiError = APIInternal
	TraceError(ctx, err)
	defer TraceAPIErrorAndFinish(ctx, w.Header(), apiError)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(apiError.Status)
	json.NewEncoder(w).Encode(apiError)
}

// APIError defines a standard format for API errors.
type APIError struct {
	// The status code.
	Status int `json:"status"`
	// The description of the API error.
	Description string `json:"description"`
	// The token uniquely identifying the API error.
	ErrorCode string `json:"errorCode"`
	// Additional infos.
	Params map[string]interface{} `json:"params,omitempty"`
}

var (
	// APIInternal indicates an unexpected internal error.
	APIInternal = APIError{
		Status:      500,
		Description: "An internal error occured. Please retry later.",
		ErrorCode:   "INTERNAL_ERROR",
		Params:      make(map[string]interface{}),
	}
	// APIUnavailable indicates that an unexpected error happened during
	// the domain execution (from go-kit). Probably caused by an overload.
	APIUnavailable = APIError{
		Status:      503,
		Description: "The service is currently unavailable. Please retry later.",
		ErrorCode:   "SERVICE_UNAVAILABLE",
		Params:      make(map[string]interface{}),
	}
	// APIBodyDecoding indicates that the request body could not be decoded (bad syntax).
	APIBodyDecoding = APIError{
		Status:      400,
		Description: "Could not decode the JSON request.",
		ErrorCode:   "BODY_DECODING_ERROR",
		Params:      make(map[string]interface{}),
	}
	// APIQueryParam indicates that an expected query parameter is missing.
	APIQueryParam = APIError{
		Status:      400,
		Description: "Missing query parameter.",
		ErrorCode:   "QUERY_PARAM_ERROR",
		Params:      make(map[string]interface{}),
	}
	// APIValidation indicates that some received parameters are invalid.
	APIValidation = APIError{
		Status:      400,
		Description: "The parameters validation failed.",
		ErrorCode:   "VALIDATION_ERROR",
		Params:      make(map[string]interface{}),
	}
	// APIUnauthorized indicates that the user does not have a valid associated session.
	APIUnauthorized = APIError{
		Status:      401,
		Description: "Authorization Required.",
		ErrorCode:   "AUTHORIZATION_REQUIRED",
		Params:      make(map[string]interface{}),
	}
	// APIForbidden indicates that the user has a valid session but he is missing some permissions.
	APIForbidden = APIError{
		Status:      403,
		Description: "The specified resource was not found or you don't have sufficient permissions.",
		ErrorCode:   "FORBIDDEN",
		Params:      make(map[string]interface{}),
	}
)
