package grpc

import (
	"context"

	"github.com/wa-labs/styx/pb"
)

// Service represents the session management service interface.
type Service interface {
	CreateSession(ctx context.Context, session *pb.Session) (*pb.Session, error)
	FindSessionByToken(ctx context.Context, token string) (*pb.Session, error)
	DeleteSessionByToken(ctx context.Context, token string) (*pb.Session, error)
	DeleteSessionsByOwnerToken(ctx context.Context, ownerToken string) ([]*pb.Session, error)
}
