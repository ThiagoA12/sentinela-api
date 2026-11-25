//go:build integration

package repository

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

func setupUsers(t *testing.T) *UserRepository {
	t.Helper()
	if _, err := testPool.Exec(t.Context(), "TRUNCATE audit_log, assets, users"); err != nil {
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

func TestUserRepositoryGetByEmail(t *testing.T) {
	repo := setupUsers(t)
	created, err := repo.Create(t.Context(), sampleUser("Ana@Exemplo.com"))
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	got, err := repo.GetByEmail(t.Context(), "ana@exemplo.COM")
	if err != nil {
		t.Fatalf("buscar com caixa diferente: %v", err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("GetByEmail = %+v\nquero %+v", got, created)
	}

	_, err = repo.GetByEmail(t.Context(), "outra@exemplo.com")
	if apperror.KindOf(err) != apperror.KindNotFound {
		t.Errorf("e-mail inexistente: quero KindNotFound, veio %v", err)
	}
}

func TestUserRepositoryGetByID(t *testing.T) {
	repo := setupUsers(t)
	created, err := repo.Create(t.Context(), sampleUser("ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	got, err := repo.GetByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("GetByID = %+v\nquero %+v", got, created)
	}

	_, err = repo.GetByID(t.Context(), uuid.New())
	if apperror.KindOf(err) != apperror.KindNotFound {
		t.Errorf("id inexistente: quero KindNotFound, veio %v", err)
	}
}

func TestUserRepositoryLoginFailures(t *testing.T) {
	repo := setupUsers(t)
	created, err := repo.Create(t.Context(), sampleUser("ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	lockedUntil := time.Date(2026, 11, 22, 19, 15, 0, 0, time.UTC)

	fail := func() domain.User {
		t.Helper()
		if err := repo.RecordLoginFailure(t.Context(), created.ID, 5, lockedUntil); err != nil {
			t.Fatalf("registrar falha: %v", err)
		}
		u, err := repo.GetByEmail(t.Context(), created.Email)
		if err != nil {
			t.Fatalf("buscar: %v", err)
		}
		return u
	}

	var u domain.User
	for range 4 {
		u = fail()
	}
	if u.FailedLogins != 4 || u.LockedUntil != nil {
		t.Fatalf("depois de 4 falhas: failed_logins %d, locked_until %v; quero 4 e nil", u.FailedLogins, u.LockedUntil)
	}

	u = fail()
	if u.FailedLogins != 0 || u.LockedUntil == nil || !u.LockedUntil.Equal(lockedUntil) {
		t.Fatalf("depois de 5 falhas: failed_logins %d, locked_until %v; quero 0 e %v", u.FailedLogins, u.LockedUntil, lockedUntil)
	}
	if u.Version != created.Version || !u.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("falha de login nao deveria mudar version nem updated_at: %d %v", u.Version, u.UpdatedAt)
	}

	if err := repo.ResetLoginFailures(t.Context(), created.ID); err != nil {
		t.Fatalf("zerar: %v", err)
	}
	u, err = repo.GetByEmail(t.Context(), created.Email)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if u.FailedLogins != 0 || u.LockedUntil != nil {
		t.Errorf("depois de zerar: failed_logins %d, locked_until %v; quero 0 e nil", u.FailedLogins, u.LockedUntil)
	}
}

func TestUserRepositoryLoginFailuresConcurrent(t *testing.T) {
	repo := setupUsers(t)
	created, err := repo.Create(t.Context(), sampleUser("ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	const attempts = 20
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := repo.RecordLoginFailure(t.Context(), created.ID, 100, time.Now()); err != nil {
				t.Errorf("registrar falha: %v", err)
			}
		}()
	}
	wg.Wait()

	u, err := repo.GetByEmail(t.Context(), created.Email)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if u.FailedLogins != attempts {
		t.Errorf("failed_logins = %d depois de %d falhas ao mesmo tempo", u.FailedLogins, attempts)
	}
}
