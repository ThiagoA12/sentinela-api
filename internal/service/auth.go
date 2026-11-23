package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/auth"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

const (
	maxLoginFailures = 5
	lockDuration     = 15 * time.Minute
)

var errInvalidCredentials = apperror.Unauthorized("credenciais invalidas")

type authStore interface {
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	RecordLoginFailure(ctx context.Context, id uuid.UUID, maxFailures int16, lockedUntil time.Time) error
	ResetLoginFailures(ctx context.Context, id uuid.UUID) error
}

type AuthService struct {
	store     authStore
	tokens    *auth.TokenManager
	now       func() time.Time
	dummyHash string
}

func NewAuthService(store authStore, tokens *auth.TokenManager, now func() time.Time) *AuthService {
	return &AuthService{
		store:     store,
		tokens:    tokens,
		now:       now,
		dummyHash: auth.HashPassword("senha de usuario inexistente"),
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.store.GetByEmail(ctx, strings.TrimSpace(email))
	if apperror.KindOf(err) == apperror.KindNotFound {
		_, _ = auth.VerifyPassword(password, s.dummyHash)
		return "", errInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	ok, err := auth.VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return "", err
	}

	now := s.now()
	if !u.Active || (u.LockedUntil != nil && now.Before(*u.LockedUntil)) {
		return "", errInvalidCredentials
	}
	if !ok {
		if err := s.store.RecordLoginFailure(ctx, u.ID, maxLoginFailures, now.Add(lockDuration)); err != nil {
			return "", err
		}
		return "", errInvalidCredentials
	}

	if u.FailedLogins > 0 || u.LockedUntil != nil {
		if err := s.store.ResetLoginFailures(ctx, u.ID); err != nil {
			return "", err
		}
	}
	return s.tokens.Issue(u.ID, now), nil
}
