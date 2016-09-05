package account

import (
	"encoding/json"
	"net/http"

	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/pressly/chi"
	"github.com/solher/styx/helpers"
	"github.com/solher/styx/sessions"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

type errBodyDecoding struct {
	helpers.ErrBehavior
	helpers.ErrBodyDecodingBehavior
}

func newErrBodyDecoding(msg string) (err errBodyDecoding) {
	defer func() { err.Msg = msg }()
	return errBodyDecoding{}
}

type errQueryParam struct {
	helpers.ErrBehavior
	helpers.ErrQueryParamBehavior
}

func newErrQueryParam(msg, key string) (err errQueryParam) {
	defer func() {
		err.Msg = msg
		err.ParamKey = key
	}()
	return errQueryParam{}
}

// MakeHTTPHandler returns a handler that makes a set of endpoints available
// on predefined paths.
func MakeHTTPHandler(ctx context.Context, endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(helpers.TransportErrorEncoder),
		httptransport.ServerErrorLogger(logger),
	}

	createSessionHandler := httptransport.NewServer(
		ctx,
		endpoints.CreateSessionEndpoint,
		DecodeHTTPCreateSessionRequest,
		EncodeHTTPCreateSessionResponse,
		append(options, httptransport.ServerBefore(helpers.FromHTTPRequest(tracer, "Create session", logger)))...,
	)
	findSessionByTokenHandler := httptransport.NewServer(
		ctx,
		endpoints.FindSessionByTokenEndpoint,
		DecodeHTTPFindSessionByTokenRequest,
		EncodeHTTPFindSessionByTokenResponse,
		append(options, httptransport.ServerBefore(helpers.FromHTTPRequest(tracer, "Find session by token", logger)))...,
	)
	deleteSessionByTokenHandler := httptransport.NewServer(
		ctx,
		endpoints.DeleteSessionByTokenEndpoint,
		DecodeHTTPDeleteSessionByTokenRequest,
		EncodeHTTPDeleteSessionByTokenResponse,
		append(options, httptransport.ServerBefore(helpers.FromHTTPRequest(tracer, "Delete session by token", logger)))...,
	)
	deleteSessionByOwnerTokenHandler := httptransport.NewServer(
		ctx,
		endpoints.DeleteSessionsByOwnerTokenEndpoint,
		DecodeHTTPDeleteSessionsByOwnerTokenRequest,
		EncodeHTTPDeleteSessionsByOwnerTokenResponse,
		append(options, httptransport.ServerBefore(helpers.FromHTTPRequest(tracer, "Delete session by owner token", logger)))...,
	)

	r := chi.NewRouter()
	r.Route("/sessions", func(r chi.Router) {
		r.Post("/", createSessionHandler.ServeHTTP)
		r.Get("/:token", findSessionByTokenHandler.ServeHTTP)
		r.Delete("/:token", deleteSessionByTokenHandler.ServeHTTP)
		r.Delete("/", deleteSessionByOwnerTokenHandler.ServeHTTP)
	})

	return r
}

// DecodeHTTPCreateSessionRequest is a transport/http.DecodeRequestFunc that decodes the
// JSON-encoded request from the HTTP request body.
func DecodeHTTPCreateSessionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	session := &sessions.Session{}
	if err := json.NewDecoder(r.Body).Decode(session); err != nil {
		return nil, errors.Wrap(newErrBodyDecoding(err.Error()), "could not decode the session")
	}
	return createSessionRequest{
		Session: session,
	}, nil
}

// EncodeHTTPCreateSessionResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer.
func EncodeHTTPCreateSessionResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(createSessionResponse)
	if res.Err != nil {
		return businessErrorEncoder(ctx, res.Err, w)
	}
	defer helpers.TraceStatusAndFinish(ctx, 201)
	encodeSession(w, res.Session, 201)
	return nil
}

// DecodeHTTPFindSessionByTokenRequest is a transport/http.DecodeRequestFunc that decodes the
// JSON-encoded request from the HTTP request body.
func DecodeHTTPFindSessionByTokenRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return findSessionByTokenRequest{
		Token: chi.URLParam(r, "token"),
	}, nil
}

// EncodeHTTPFindSessionByTokenResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer.
func EncodeHTTPFindSessionByTokenResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(findSessionByTokenResponse)
	if res.Err != nil {
		return businessErrorEncoder(ctx, res.Err, w)
	}
	defer helpers.TraceStatusAndFinish(ctx, 200)
	encodeSession(w, res.Session, 200)
	return nil
}

// DecodeHTTPDeleteSessionByTokenRequest is a transport/http.DecodeRequestFunc that decodes the
// JSON-encoded request from the HTTP request body.
func DecodeHTTPDeleteSessionByTokenRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return deleteSessionByTokenRequest{
		Token: chi.URLParam(r, "token"),
	}, nil
}

// EncodeHTTPDeleteSessionByTokenResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer.
func EncodeHTTPDeleteSessionByTokenResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(deleteSessionByTokenResponse)
	if res.Err != nil {
		return businessErrorEncoder(ctx, res.Err, w)
	}
	defer helpers.TraceStatusAndFinish(ctx, 200)
	encodeSession(w, res.Session, 200)
	return nil
}

// DecodeHTTPDeleteSessionsByOwnerTokenRequest is a transport/http.DecodeRequestFunc that decodes the
// JSON-encoded request from the HTTP request body.
func DecodeHTTPDeleteSessionsByOwnerTokenRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	values, ok := r.URL.Query()["ownerToken"]
	if !ok {
		return nil, errors.Wrap(newErrQueryParam("ownerToken parameter is required", "ownerToken"), "decoding failed")
	}
	return deleteSessionsByOwnerTokenRequest{
		OwnerToken: values[0],
	}, nil
}

// EncodeHTTPDeleteSessionsByOwnerTokenResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer.
func EncodeHTTPDeleteSessionsByOwnerTokenResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(deleteSessionsByOwnerTokenResponse)
	if res.Err != nil {
		return businessErrorEncoder(ctx, res.Err, w)
	}
	defer helpers.TraceStatusAndFinish(ctx, 200)
	encodeSessions(w, res.Sessions, 200)
	return nil
}

type (
	errValidationType interface {
		error
		IsErrValidation()
		Field() string
		Reason() string
	}
	errValidationBehavior struct {
		field, reason string
	}
)

func (err errValidationBehavior) IsErrValidation() {}
func (err errValidationBehavior) Field() string    { return err.field }
func (err errValidationBehavior) Reason() string   { return err.reason }

type (
	errNotFoundType interface {
		error
		IsErrNotFound()
	}
	errNotFoundBehavior struct{}
)

func (err errNotFoundBehavior) IsErrNotFound() {}

func businessErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) error {
	var apiError helpers.APIError
	switch err := errors.Cause(err).(type) {
	case errValidationType:
		apiError = helpers.APIValidation
		apiError.Params["field"] = err.Field()
		apiError.Params["reason"] = err.Reason()
	case errNotFoundType:
		apiError = helpers.APIForbidden
	default:
		return err
	}
	defer helpers.TraceAPIErrorAndFinish(ctx, apiError)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(apiError.Status)
	json.NewEncoder(w).Encode(apiError)
	return nil
}

func encodeSession(w http.ResponseWriter, session *sessions.Session, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(session)
}

func encodeSessions(w http.ResponseWriter, sessions []sessions.Session, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(sessions)
}
