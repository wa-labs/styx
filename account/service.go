package account

import (
	"context"

	"github.com/pkg/errors"
	"github.com/solher/styx/helpers"
	"github.com/solher/styx/sessions"
)

// ErrValidation is returned when the parameters validation failed.
type ErrValidation struct {
	helpers.BasicError
	Field, Reason string
}

// NewErrValidation returns a new instance of ErrValidation.
func NewErrValidation(msg, field, reason string) ErrValidation {
	return ErrValidation{
		BasicError: helpers.NewBasicError(msg),
		Field:      field,
		Reason:     reason,
	}
}

// Service represents the account service interface.
type Service interface {
	CreateSession(ctx context.Context, session *sessions.Session) (*sessions.Session, error)
	FindSessionByToken(ctx context.Context, token string) (*sessions.Session, error)
	DeleteSessionByToken(ctx context.Context, token string) (*sessions.Session, error)
	DeleteSessionsByOwnerToken(ctx context.Context, ownerToken string) ([]sessions.Session, error)
}

type service struct {
	sessionRepo sessions.Repository
}

// NewService returns a new instance of the account service.
func NewService(sessionRepo sessions.Repository) Service {
	return &service{
		sessionRepo: sessionRepo,
	}
}

// CreateSession creates a new session.
func (s *service) CreateSession(ctx context.Context, session *sessions.Session) (*sessions.Session, error) {
	if session.Policies == nil {
		return nil, errors.Wrap(NewErrValidation("session policies cannot be blank", "policies", "blank"), "validation failed")
	}
	return s.sessionRepo.Create(ctx, session)
}

// FindSessionByToken finds a session by its token and returns it.
func (s *service) FindSessionByToken(ctx context.Context, token string) (*sessions.Session, error) {
	return s.sessionRepo.FindByToken(ctx, token)
}

// DeleteSessionByToken deletes a session by its token and returns it.
func (s *service) DeleteSessionByToken(ctx context.Context, token string) (*sessions.Session, error) {
	return s.sessionRepo.DeleteByToken(ctx, token)
}

// DeleteSessionsByOwnerToken deletes all the sessions having the same ownerToken.
// Useful to implement delete cascades on user deletions.
func (s *service) DeleteSessionsByOwnerToken(ctx context.Context, ownerToken string) ([]sessions.Session, error) {
	return s.sessionRepo.DeleteByOwnerToken(ctx, ownerToken)
}
