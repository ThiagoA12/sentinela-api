package repository

import (
	"context"
	"fmt"

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
