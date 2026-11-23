package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/auth"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

const (
	testUserPassword = "senha-certa-do-usuario"
	testTokenKey     = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
)

var (
	testUserHash = auth.HashPassword(testUserPassword)
	loginNow     = time.Date(2026, 11, 22, 19, 0, 0, 0, time.UTC)
	loginUserID  = uuid.MustParse("01a1cab1-9c80-7a10-8000-000000000001")
)

type loginFailure struct {
	id          uuid.UUID
	maxFailures int16
	lockedUntil time.Time
}

type fakeAuthStore struct {
	user     *domain.User
	getErr   error
	failErr  error
	resetErr error
	gotEmail string
	failures []loginFailure
	resets   int
}

func (f *fakeAuthStore) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	f.gotEmail = email
	if f.getErr != nil {
		return domain.User{}, f.getErr
	}
	if f.user == nil {
		return domain.User{}, apperror.NotFound("usuario nao encontrado")
	}
	return *f.user, nil
}

func (f *fakeAuthStore) RecordLoginFailure(ctx context.Context, id uuid.UUID, maxFailures int16, lockedUntil time.Time) error {
	f.failures = append(f.failures, loginFailure{id, maxFailures, lockedUntil})
	return f.failErr
}

func (f *fakeAuthStore) ResetLoginFailures(ctx context.Context, id uuid.UUID) error {
	f.resets++
	return f.resetErr
}

func activeUser() *domain.User {
	return &domain.User{
		ID:           loginUserID,
		Email:        "ana@exemplo.com",
		PasswordHash: testUserHash,
		Role:         domain.RoleAnalista,
		Active:       true,
	}
}

func newAuthService(t *testing.T, store *fakeAuthStore) (*service.AuthService, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(testTokenKey)
	if err != nil {
		t.Fatalf("chave de teste: %v", err)
	}
	return service.NewAuthService(store, tokens, func() time.Time { return loginNow }), tokens
}

func TestLogin(t *testing.T) {
	past := loginNow.Add(-time.Second)

	tests := []struct {
		name         string
		failedLogins int16
		lockedUntil  *time.Time
		wantResets   int
	}{
		{"sem falhas", 0, nil, 0},
		{"com falhas antes", 3, nil, 1},
		{"bloqueio vencido", 0, &past, 1},
		{"bloqueio vence agora", 0, &loginNow, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := activeUser()
			u.FailedLogins = tt.failedLogins
			u.LockedUntil = tt.lockedUntil
			store := &fakeAuthStore{user: u}
			svc, tokens := newAuthService(t, store)

			token, err := svc.Login(t.Context(), "  ana@exemplo.com\n", testUserPassword)
			if err != nil {
				t.Fatalf("Login: %v", err)
			}
			id, err := tokens.Verify(token, loginNow)
			if err != nil || id != loginUserID {
				t.Errorf("token do Login: id %v, erro %v; quero %v", id, err, loginUserID)
			}
			if store.gotEmail != "ana@exemplo.com" {
				t.Errorf("e-mail buscado = %q, quero sem espacos", store.gotEmail)
			}
			if len(store.failures) != 0 {
				t.Errorf("login certo registrou %d falha(s)", len(store.failures))
			}
			if store.resets != tt.wantResets {
				t.Errorf("ResetLoginFailures chamado %d vez(es), quero %d", store.resets, tt.wantResets)
			}
		})
	}
}

func TestLoginRecusado(t *testing.T) {
	future := loginNow.Add(time.Second)

	tests := []struct {
		name         string
		missing      bool
		change       func(u *domain.User)
		password     string
		wantFailures int
	}{
		{"e-mail inexistente", true, nil, testUserPassword, 0},
		{"senha errada", false, nil, "senha-errada-do-usuario", 1},
		{"senha com espaco a mais", false, nil, testUserPassword + " ", 1},
		{"desativado com a senha certa", false, func(u *domain.User) { u.Active = false }, testUserPassword, 0},
		{"bloqueado com a senha certa", false, func(u *domain.User) { u.LockedUntil = &future }, testUserPassword, 0},
		{"bloqueado com a senha errada", false, func(u *domain.User) { u.LockedUntil = &future }, "senha-errada-do-usuario", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeAuthStore{user: activeUser()}
			if tt.change != nil {
				tt.change(store.user)
			}
			if tt.missing {
				store.user = nil
			}
			svc, _ := newAuthService(t, store)

			token, err := svc.Login(t.Context(), "ana@exemplo.com", tt.password)
			if token != "" {
				t.Errorf("login recusado devolveu token %q", token)
			}
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Kind != apperror.KindUnauthorized || appErr.Detail != "credenciais invalidas" {
				t.Fatalf("quero Unauthorized \"credenciais invalidas\", veio %v", err)
			}
			if len(store.failures) != tt.wantFailures {
				t.Fatalf("%d falha(s) registrada(s), quero %d", len(store.failures), tt.wantFailures)
			}
			if tt.wantFailures == 1 {
				want := loginFailure{loginUserID, 5, loginNow.Add(15 * time.Minute)}
				if store.failures[0] != want {
					t.Errorf("RecordLoginFailure recebeu %+v, quero %+v", store.failures[0], want)
				}
			}
			if store.resets != 0 {
				t.Errorf("login recusado zerou o contador")
			}
		})
	}
}

func TestLoginErroDoStore(t *testing.T) {
	dbErr := errors.New("banco fora do ar")
	withFailures := activeUser()
	withFailures.FailedLogins = 2
	brokenHash := activeUser()
	brokenHash.PasswordHash = "hash-quebrado"

	tests := []struct {
		name     string
		store    *fakeAuthStore
		password string
		want     error
	}{
		{"erro na busca", &fakeAuthStore{getErr: dbErr}, testUserPassword, dbErr},
		{"erro ao registrar falha", &fakeAuthStore{user: activeUser(), failErr: dbErr}, "senha-errada-do-usuario", dbErr},
		{"erro ao zerar", &fakeAuthStore{user: withFailures, resetErr: dbErr}, testUserPassword, dbErr},
		{"hash quebrado no banco", &fakeAuthStore{user: brokenHash}, testUserPassword, auth.ErrInvalidHash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newAuthService(t, tt.store)

			token, err := svc.Login(t.Context(), "ana@exemplo.com", tt.password)
			if token != "" || !errors.Is(err, tt.want) {
				t.Errorf("Login = %q, %v; quero \"\", %v", token, err, tt.want)
			}
		})
	}
}

func TestLoginTempoParecido(t *testing.T) {
	measure := func(t *testing.T, user *domain.User) time.Duration {
		t.Helper()
		svc, _ := newAuthService(t, &fakeAuthStore{user: user})
		best := time.Hour
		for range 3 {
			start := time.Now()
			_, _ = svc.Login(t.Context(), "ana@exemplo.com", "senha-errada-do-usuario")
			best = min(best, time.Since(start))
		}
		return best
	}

	future := loginNow.Add(time.Second)
	inactive := activeUser()
	inactive.Active = false
	locked := activeUser()
	locked.LockedUntil = &future

	wrongPassword := measure(t, activeUser())

	tests := []struct {
		name string
		user *domain.User
	}{
		{"e-mail inexistente", nil},
		{"desativado", inactive},
		{"bloqueado", locked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if d := measure(t, tt.user); d < wrongPassword/2 {
				t.Errorf("levou %v, a senha errada levou %v: o tempo denuncia a diferenca", d, wrongPassword)
			}
		})
	}
}
