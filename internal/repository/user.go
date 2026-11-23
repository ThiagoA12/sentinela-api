package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/db"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(conn db.DBTX) *UserRepository {
	return &UserRepository{q: db.New(conn)}
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	n, err := r.q.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("contar usuarios: %w", err)
	}
	return n, nil
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("criar usuario: %w", emailInUse(err))
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, apperror.NotFound("usuario nao encontrado")
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("buscar usuario por email: %w", err)
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) RecordLoginFailure(ctx context.Context, id uuid.UUID, maxFailures int16, lockedUntil time.Time) error {
	err := r.q.RecordLoginFailure(ctx, db.RecordLoginFailureParams{
		MaxFailures: maxFailures,
		LockedUntil: lockedUntil,
		ID:          id,
	})
	if err != nil {
		return fmt.Errorf("registrar falha de login: %w", err)
	}
	return nil
}

func (r *UserRepository) ResetLoginFailures(ctx context.Context, id uuid.UUID) error {
	if err := r.q.ResetLoginFailures(ctx, id); err != nil {
		return fmt.Errorf("zerar falhas de login: %w", err)
	}
	return nil
}

func toDomainUser(u db.User) domain.User {
	return domain.User{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		Active:       u.Active,
		FailedLogins: u.FailedLogins,
		LockedUntil:  u.LockedUntil,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Version:      u.Version,
	}
}
