package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

type fakeUserService struct {
	user   domain.User
	err    error
	called bool
	in     service.CreateUserInput
}

func (f *fakeUserService) Create(ctx context.Context, in service.CreateUserInput) (domain.User, error) {
	f.called = true
	f.in = in
	return f.user, f.err
}

func postUsuario(svc *fakeUserService, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/usuarios", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handleCriaUsuario(svc)(rec, req)
	return rec
}

func TestHandleCriaUsuario(t *testing.T) {
	svc := &fakeUserService{user: domain.User{
		ID:           uuid.MustParse("01a1cab1-9c80-7a10-8000-000000000002"),
		Name:         "Ana",
		Email:        "ana@exemplo.com",
		PasswordHash: "$argon2id$hash-que-nunca-sai",
		Role:         domain.RoleAnalista,
		Active:       true,
	}}

	rec := postUsuario(svc, "application/json", `{"name":" Ana ","email":" ana@exemplo.com","password":" senha com espacos ","role":"analista"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado 201; corpo: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, esperado application/json", got)
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Errorf("Location = %q, esperado nenhum: nao existe GET /usuarios/{id}", got)
	}
	want := `{"id":"01a1cab1-9c80-7a10-8000-000000000002","name":"Ana","email":"ana@exemplo.com","role":"analista"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("corpo = %s, esperado %s", got, want)
	}
	wantIn := service.CreateUserInput{Name: " Ana ", Email: " ana@exemplo.com", Password: " senha com espacos ", Role: "analista"}
	if svc.in != wantIn {
		t.Errorf("service recebeu %+v; o handler nao deveria mexer nos valores", svc.in)
	}
}

func TestHandleCriaUsuarioRequisicaoRecusada(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantDetail  string
	}{
		{"text plain", "text/plain", `{"name":"Ana"}`, http.StatusUnsupportedMediaType, "Content-Type deve ser application/json"},
		{"json cortado", "application/json", `{"name":`, http.StatusBadRequest, "JSON malformado"},
		{"campo active", "application/json", `{"name":"Ana","email":"ana@exemplo.com","password":"senha-longa-de-teste","role":"leitor","active":false}`, http.StatusBadRequest, `campo desconhecido: "active"`},
		{"campo password_hash", "application/json", `{"name":"Ana","email":"ana@exemplo.com","password_hash":"$argon2id$v=19$forjado","role":"admin"}`, http.StatusBadRequest, `campo desconhecido: "password_hash"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{}
			rec := postUsuario(svc, tt.contentType, tt.body)

			var p struct {
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e JSON: %v", err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if svc.called {
				t.Error("service nao deveria ter sido chamado")
			}
		})
	}
}

func TestHandleCriaUsuarioErroDoService(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
	}{
		{"e-mail repetido", apperror.Conflict("ja existe um usuario com esse email", errors.New("23505")), http.StatusConflict, "ja existe um usuario com esse email"},
		{"campo invalido", apperror.Validation([]apperror.FieldError{{Field: "role", Reason: "deve ser: admin, analista ou leitor"}}), http.StatusUnprocessableEntity, "campo(s) invalidos"},
		{"erro interno", errors.New("banco fora do ar"), http.StatusInternalServerError, "erro inesperado. tente novamente mais tarde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{err: tt.err}
			rec := postUsuario(svc, "application/json", `{"name":"Ana","email":"ana@exemplo.com","password":"senha-longa-de-teste","role":"analista"}`)

			var p struct {
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e JSON: %v", err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if strings.Contains(rec.Body.String(), "senha-longa-de-teste") {
				t.Errorf("resposta de erro ecoou a senha: %s", rec.Body.String())
			}
		})
	}
}
