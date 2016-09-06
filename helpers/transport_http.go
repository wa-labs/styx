package helpers

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
)

type (
	errBodyDecoding interface {
		error
		IsErrBodyDecoding()
	}
	errQueryParam interface {
		error
		IsErrQueryParam()
		Key() string
	}
)

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

type errQueryParamBehavior struct {
	key string
}

func (e errQueryParamBehavior) IsErrQueryParam() {}
func (e errQueryParamBehavior) Key() string      { return e.key }

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

func TransportErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	var apiError APIError
	switch e1 := err.(type) {
	case httptransport.Error:
		err = e1.Err
		switch e1.Domain {
		case httptransport.DomainDecode:
			switch e2 := errors.Cause(err).(type) {
			case errBodyDecoding:
				apiError = APIBodyDecoding
			case errQueryParam:
				apiError = APIQueryParam
				apiError.Params["key"] = e2.Key()
			default:
				apiError = APIInternal
				TraceError(ctx, err)
			}
		case httptransport.DomainDo:
			apiError = APIUnavailable
			TraceError(ctx, err)
		default:
			apiError = APIInternal
			TraceError(ctx, err)
		}
	default:
		apiError = APIInternal
		TraceError(ctx, err)
	}
	defer TraceAPIErrorAndFinish(ctx, apiError)

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
	APIInternal = APIError{
		Status:      500,
		Description: "An internal error occured. Please retry later.",
		ErrorCode:   "INTERNAL_ERROR",
		Params:      make(map[string]interface{}),
	}
	APIUnavailable = APIError{
		Status:      503,
		Description: "The service is currently unavailable. Please retry later.",
		ErrorCode:   "SERVICE_UNAVAILABLE",
		Params:      make(map[string]interface{}),
	}
	APIBodyDecoding = APIError{
		Status:      400,
		Description: "Could not decode the JSON request.",
		ErrorCode:   "BODY_DECODING_ERROR",
		Params:      make(map[string]interface{}),
	}
	APIQueryParam = APIError{
		Status:      400,
		Description: "Missing query parameter.",
		ErrorCode:   "QUERY_PARAM_ERROR",
		Params:      make(map[string]interface{}),
	}
	APIValidation = APIError{
		Status:      400,
		Description: "The parameters validation failed.",
		ErrorCode:   "VALIDATION_ERROR",
		Params:      make(map[string]interface{}),
	}
	APIUnauthorized = APIError{
		Status:      401,
		Description: "Authorization Required.",
		ErrorCode:   "AUTHORIZATION_REQUIRED",
		Params:      make(map[string]interface{}),
	}
	APIForbidden = APIError{
		Status:      403,
		Description: "The specified resource was not found or you don't have sufficient permissions.",
		ErrorCode:   "FORBIDDEN",
		Params:      make(map[string]interface{}),
	}
)
