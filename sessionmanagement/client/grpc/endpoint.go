package grpc

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/wa-labs/styx/pb"
)

// Endpoints collects all of the endpoints that compose a session management service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	CreateSessionEndpoint              endpoint.Endpoint
	FindSessionByTokenEndpoint         endpoint.Endpoint
	DeleteSessionByTokenEndpoint       endpoint.Endpoint
	DeleteSessionsByOwnerTokenEndpoint endpoint.Endpoint
}

// CreateSession implements Service.
func (e Endpoints) CreateSession(ctx context.Context, session *pb.Session) (*pb.Session, error) {
	req := &pb.CreateSessionRequest{
		Session: session,
	}
	response, err := e.CreateSessionEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}
	res := response.(*pb.CreateSessionReply)
	return res.Session, toError(res.Err)
}

// FindSessionByToken implements Service.
func (e Endpoints) FindSessionByToken(ctx context.Context, token string) (*pb.Session, error) {
	req := &pb.FindSessionByTokenRequest{
		Token: token,
	}
	response, err := e.FindSessionByTokenEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}
	res := response.(*pb.FindSessionByTokenReply)
	return res.Session, toError(res.Err)
}

// DeleteSessionByToken implements Service.
func (e Endpoints) DeleteSessionByToken(ctx context.Context, token string) (*pb.Session, error) {
	req := &pb.DeleteSessionByTokenRequest{
		Token: token,
	}
	response, err := e.DeleteSessionByTokenEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}
	res := response.(*pb.DeleteSessionByTokenReply)
	return res.Session, toError(res.Err)
}

// DeleteSessionsByOwnerToken implements Service.
func (e Endpoints) DeleteSessionsByOwnerToken(ctx context.Context, ownerToken string) ([]*pb.Session, error) {
	req := &pb.DeleteSessionsByOwnerTokenRequest{
		OwnerToken: ownerToken,
	}
	response, err := e.DeleteSessionsByOwnerTokenEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}
	res := response.(*pb.DeleteSessionsByOwnerTokenReply)
	return res.Sessions, toError(res.Err)
}

func toError(err string) error {
	if len(err) == 0 {
		return nil
	}
	return errors.New(err)
}
