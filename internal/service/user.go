package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ThiagoA12/sentinela-api/internal/auth"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

const (
	maxEmailLen      = 254
	initialAdminName = "Administrador"
)

type userStore interface {
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, u domain.User) (domain.User, error)
}

type UserService struct {
	store userStore
}

func NewUserService(store userStore) *UserService {
	return &UserService{store: store}
}

func (s *UserService) EnsureInitialAdmin(ctx context.Context, email, password string) (bool, error) {
	n, err := s.store.Count(ctx)
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}

	email = strings.TrimSpace(email)
	if !validEmail(email) {
		return false, errors.New("ADMIN_EMAIL invalido: use nome@dominio, max 254 caracteres")
	}
	if !auth.ValidPasswordLength(password) {
		return false, fmt.Errorf("ADMIN_PASSWORD deve ter entre %d e %d caracteres", auth.MinPasswordLength, auth.MaxPasswordLength)
	}

	_, err = s.store.Create(ctx, domain.User{
		Name:         initialAdminName,
		Email:        email,
		PasswordHash: auth.HashPassword(password),
		Role:         domain.RoleAdmin,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func validEmail(email string) bool {
	if utf8.RuneCountInString(email) > maxEmailLen {
		return false
	}
	local, host, ok := strings.Cut(email, "@")
	return ok && local != "" && host != "" && !strings.Contains(host, "@")
}
