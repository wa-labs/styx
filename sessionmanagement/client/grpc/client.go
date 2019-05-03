package grpc

import (
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/wa-labs/styx/kitty"
	"github.com/wa-labs/styx/pb"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

var noop = func(_ context.Context, r interface{}) (interface{}, error) {
	return r, nil
}

// New returns a session management service backed by a gRPC client connection. It is the
// responsibility of the caller to dial, and later close, the connection.
func New(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger, opts ...Option) Service {
	clientOpts := &options{
		limiterRate:     100,
		limiterCapacity: 100,
	}
	for _, opt := range opts {
		opt(clientOpts)
	}
	limit := rate.NewLimiter(rate.Every(time.Minute), 100)
	limiter := ratelimit.NewDelayingLimiter(limit)
	// limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(clientOpts.limiterRate, clientOpts.limiterCapacity))
	circuitbreaker := circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "SessionManagement"}))
	var createSessionEndpoint endpoint.Endpoint
	{
		createSessionEndpoint = grpctransport.NewClient(
			conn,
			"SessionManagement",
			"CreateSession",
			noop,
			noop,
			pb.CreateSessionReply{},
			grpctransport.ClientBefore(kitty.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		createSessionEndpoint = opentracing.TraceClient(tracer, "Create session")(createSessionEndpoint)
		createSessionEndpoint = kitty.EndpointTracingMiddleware(createSessionEndpoint)
		createSessionEndpoint = limiter(createSessionEndpoint)
		createSessionEndpoint = circuitbreaker(createSessionEndpoint)
	}
	var findSessionByTokenEndpoint endpoint.Endpoint
	{
		findSessionByTokenEndpoint = grpctransport.NewClient(
			conn,
			"SessionManagement",
			"FindSessionByToken",
			noop,
			noop,
			pb.FindSessionByTokenReply{},
			grpctransport.ClientBefore(kitty.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		findSessionByTokenEndpoint = opentracing.TraceClient(tracer, "Find session by token")(findSessionByTokenEndpoint)
		findSessionByTokenEndpoint = kitty.EndpointTracingMiddleware(findSessionByTokenEndpoint)
		findSessionByTokenEndpoint = limiter(findSessionByTokenEndpoint)
		findSessionByTokenEndpoint = circuitbreaker(findSessionByTokenEndpoint)
	}
	var deleteSessionByTokenEndpoint endpoint.Endpoint
	{
		deleteSessionByTokenEndpoint = grpctransport.NewClient(
			conn,
			"SessionManagement",
			"DeleteSessionByToken",
			noop,
			noop,
			pb.DeleteSessionByTokenReply{},
			grpctransport.ClientBefore(kitty.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		deleteSessionByTokenEndpoint = opentracing.TraceClient(tracer, "Delete session by token")(deleteSessionByTokenEndpoint)
		deleteSessionByTokenEndpoint = kitty.EndpointTracingMiddleware(deleteSessionByTokenEndpoint)
		deleteSessionByTokenEndpoint = limiter(deleteSessionByTokenEndpoint)
		deleteSessionByTokenEndpoint = circuitbreaker(deleteSessionByTokenEndpoint)
	}
	var deleteSessionsByOwnerTokenEndpoint endpoint.Endpoint
	{
		deleteSessionsByOwnerTokenEndpoint = grpctransport.NewClient(
			conn,
			"SessionManagement",
			"DeleteSessionsByOwnerToken",
			noop,
			noop,
			pb.DeleteSessionsByOwnerTokenReply{},
			grpctransport.ClientBefore(kitty.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		deleteSessionsByOwnerTokenEndpoint = opentracing.TraceClient(tracer, "Delete sessions by owner token")(deleteSessionsByOwnerTokenEndpoint)
		deleteSessionsByOwnerTokenEndpoint = kitty.EndpointTracingMiddleware(deleteSessionsByOwnerTokenEndpoint)
		deleteSessionsByOwnerTokenEndpoint = limiter(deleteSessionsByOwnerTokenEndpoint)
		deleteSessionsByOwnerTokenEndpoint = circuitbreaker(deleteSessionsByOwnerTokenEndpoint)
	}

	return Endpoints{
		CreateSessionEndpoint:              createSessionEndpoint,
		FindSessionByTokenEndpoint:         findSessionByTokenEndpoint,
		DeleteSessionByTokenEndpoint:       deleteSessionByTokenEndpoint,
		DeleteSessionsByOwnerTokenEndpoint: deleteSessionsByOwnerTokenEndpoint,
	}
}

type options struct {
	limiterRate     float64
	limiterCapacity int64
}

// Option sets an optional parameter for the gRPC client.
type Option func(*options)

// LimiterRate sets the filling rate of the limiter bucket (tokens/sec).
func LimiterRate(rate float64) Option {
	return func(o *options) {
		o.limiterRate = rate
	}
}

// LimiterCapacity sets the capacity of the limiter bucket (tokens/sec).
func LimiterCapacity(capacity int64) Option {
	return func(o *options) {
		o.limiterCapacity = capacity
	}
}
