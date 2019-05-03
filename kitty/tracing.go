package kitty

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	httptransport "github.com/go-kit/kit/transport/http"

	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

// FromGRPCRequest returns a grpc RequestFunc that tries to join with an
// OpenTracing trace found in `req` and starts a new Span called
// `operationName` accordingly. If no trace could be found in `req`, the Span
// will be a trace root. The Span is incorporated in the returned Context and
// can be retrieved with opentracing.SpanFromContext(ctx).
func FromGRPCRequest(tracer stdopentracing.Tracer, operationName string, logger log.Logger) grpctransport.ServerRequestFunc {
	return func(ctx context.Context, md metadata.MD) context.Context {
		ctx = opentracing.GRPCToContext(tracer, operationName, logger)(ctx, md)
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			span = span.SetTag("transport", "gRPC")
			ctx = stdopentracing.ContextWithSpan(ctx, span)
		}
		return ctx
	}
}

// GRPCFinish finishes a span if a trace could be found.
func GRPCFinish() grpctransport.ServerResponseFunc {
	return func(ctx context.Context, header *metadata.MD, trailer *metadata.MD) context.Context {
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			span.Finish()
		}
		return ctx
	}
}

// ToGRPCRequest returns a grpc RequestFunc that injects an OpenTracing Span
// found in `ctx` into the grpc Metadata. If no such Span can be found, the
// RequestFunc is a noop.
func ToGRPCRequest(tracer stdopentracing.Tracer, logger log.Logger) grpctransport.ClientRequestFunc {
	return func(ctx context.Context, md *metadata.MD) context.Context {
		ctx = opentracing.GRPCToContext(tracer, "", logger)(ctx, *md)
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			span = span.SetTag("transport", "gRPC")
			ctx = stdopentracing.ContextWithSpan(ctx, span)
		}
		return ctx
	}
}

// FromHTTPRequest returns an http RequestFunc that tries to join with an
// OpenTracing trace found in `req` and starts a new Span called
// `operationName` accordingly. If no trace could be found in `req`, the Span
// will be a trace root. The Span is incorporated in the returned Context and
// can be retrieved with opentracing.SpanFromContext(ctx).
func FromHTTPRequest(tracer stdopentracing.Tracer, operationName string, logger log.Logger) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		ctx = opentracing.HTTPToContext(tracer, operationName, logger)(ctx, r)
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			span = span.SetTag("transport", "HTTP")
			span = span.SetTag("req.header", formatHeader(r.Header))
			span = span.SetTag("req.method", r.Method)
			span = span.SetTag("req.url", r.URL.String())
			span = span.SetTag("req.remote", r.RemoteAddr)
			ctx = stdopentracing.ContextWithSpan(ctx, span)
		}
		return ctx
	}
}

// ToHTTPResponse returns a http ServerResponseFunc that injects an OpenTracing Span
// found in `ctx` in the response HTTP header.
// Typicaly useful for tracing using subrequests. eg. Nginx auth subrequests.
func ToHTTPResponse(tracer stdopentracing.Tracer, logger log.Logger) httptransport.ServerResponseFunc {
	return func(ctx context.Context, rw http.ResponseWriter) context.Context {
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			tracer.Inject(
				span.Context(),
				stdopentracing.HTTPHeaders,
				stdopentracing.HTTPHeadersCarrier(rw.Header()),
			)
			ctx = stdopentracing.ContextWithSpan(ctx, span)
		}
		return ctx
	}
}

// TraceStatusAndFinish traces a HTTP response and finishes the span if
// a trace could be found.
func TraceStatusAndFinish(ctx context.Context, header http.Header, status int) {
	if span := stdopentracing.SpanFromContext(ctx); span != nil {
		span = span.SetTag("res.status", status)
		span = span.SetTag("res.header", formatHeader(header))
		span.Finish()
	}
}

// TraceAPIErrorAndFinish traces an APIError response and finishes the span if
// a trace could be found.
func TraceAPIErrorAndFinish(ctx context.Context, header http.Header, err APIError) {
	if span := stdopentracing.SpanFromContext(ctx); span != nil {
		if span := stdopentracing.SpanFromContext(ctx); span != nil {
			span = span.SetTag("res.status", err.Status)
			span = span.SetTag("res.header", formatHeader(header))
			span = span.SetTag("res.errorDescription", err.Description)
			span = span.SetTag("res.errorCode", err.ErrorCode)
			span = span.SetTag("res.errorParams", formatErrorParams(err.Params))
			span.Finish()
		}
	}
}

// TraceError traces an error into an 'error' tag that appears red on Zipkin.
// It is typically used to show that something unexpected happened.
// It also tags the stacktrace if it could be found.
func TraceError(ctx context.Context, err error) {
	if span := stdopentracing.SpanFromContext(ctx); span != nil {
		type stackTracer interface {
			StackTrace() errors.StackTrace
		}
		if e, ok := err.(stackTracer); ok {
			st := e.StackTrace()[0]
			split := strings.Split(fmt.Sprintf("%+v", st), "\t")
			if len(split) == 2 {
				span = span.SetTag("errorLocation", split[1])
			}
		}
		span = span.SetTag("error", err.Error())
	}
}

// EndpointTracingMiddleware traces unexpected errors at the endpoint level.
func EndpointTracingMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		defer func() {
			if err != nil {
				TraceError(ctx, err)
			}
		}()
		return next(ctx, request)
	}
}

func formatErrorParams(params map[string]interface{}) string {
	buf := bytes.NewBuffer(nil)
	for k, v := range params {
		buf.WriteString("'")
		buf.WriteString(k)
		buf.WriteString("': ")
		buf.WriteString(fmt.Sprintf("%v", v))
		buf.WriteString(", ")
	}
	if buf.Len() >= 2 {
		return buf.String()[:buf.Len()-2]
	}
	return buf.String()
}

func formatHeader(header http.Header) string {
	buf := bytes.NewBuffer(nil)
	for k, values := range header {
		for _, v := range values {
			buf.WriteString("'")
			buf.WriteString(k)
			buf.WriteString("': '")
			buf.WriteString(v)
			buf.WriteString("', ")
		}
	}
	if buf.Len() >= 2 {
		return buf.String()[:buf.Len()-2]
	}
	return buf.String()
}
