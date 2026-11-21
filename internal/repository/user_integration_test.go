//go:build integration

package repository

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

func setupUsers(t *testing.T) *UserRepository {
	t.Helper()
	if _, err := testPool.Exec(t.Context(), "TRUNCATE audit_log, users"); err != nil {
		t.Fatalf("limpar users: %v", err)
	}
	return NewUserRepository(testPool)
}

func sampleUser(email string) domain.User {
	return domain.User{
		Name:         "Usuario de teste",
		Email:        email,
		PasswordHash: "hash-de-teste",
		Role:         domain.RoleAnalista,
	}
}

func TestUserRepositoryCountCreate(t *testing.T) {
	repo := setupUsers(t)

	n, err := repo.Count(t.Context())
	if err != nil {
		t.Fatalf("contar: %v", err)
	}
	if n != 0 {
		t.Fatalf("Count = %d com a tabela vazia, quero 0", n)
	}

	in := sampleUser("ana@exemplo.com")
	created, err := repo.Create(t.Context(), in)
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	if created.ID.Version() != 7 {
		t.Errorf("id deveria ser UUIDv7, veio versao %d", created.ID.Version())
	}
	if created.CreatedAt.IsZero() || !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Errorf("datas inesperadas: created_at %v, updated_at %v", created.CreatedAt, created.UpdatedAt)
	}

	want := in
	want.ID = created.ID
	want.Active = true
	want.CreatedAt = created.CreatedAt
	want.UpdatedAt = created.UpdatedAt
	want.Version = 1
	if !reflect.DeepEqual(created, want) {
		t.Errorf("Create = %+v\nquero %+v", created, want)
	}

	n, err = repo.Count(t.Context())
	if err != nil {
		t.Fatalf("contar de novo: %v", err)
	}
	if n != 1 {
		t.Errorf("Count = %d depois de criar, quero 1", n)
	}
}

func TestUserRepositoryCreateDuplicateEmail(t *testing.T) {
	repo := setupUsers(t)
	if _, err := repo.Create(t.Context(), sampleUser("ana@exemplo.com")); err != nil {
		t.Fatalf("criar: %v", err)
	}

	tests := []struct {
		name  string
		email string
	}{
		{"mesmo e-mail", "ana@exemplo.com"},
		{"caixa diferente", "Ana@Exemplo.COM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Create(t.Context(), sampleUser(tt.email))
			if apperror.KindOf(err) != apperror.KindConflict {
				t.Fatalf("quero KindConflict, veio %v", err)
			}
		})
	}
}

func TestUserRepositoryConstraints(t *testing.T) {
	repo := setupUsers(t)

	tests := []struct {
		name     string
		change   func(u *domain.User)
		wantCode string
	}{
		{"papel fora da lista", func(u *domain.User) { u.Role = "root" }, "23514"},
		{"nome com 101 caracteres", func(u *domain.User) { u.Name = strings.Repeat("a", 101) }, "22001"},
		{"e-mail com 255 caracteres", func(u *domain.User) { u.Email = strings.Repeat("a", 243) + "@exemplo.com" }, "22001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := sampleUser("ana@exemplo.com")
			tt.change(&u)

			_, err := repo.Create(t.Context(), u)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.wantCode {
				t.Fatalf("quero erro %s do postgres, veio %v", tt.wantCode, err)
			}
		})
	}
}
