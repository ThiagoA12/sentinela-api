package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

type fakeAuthService struct {
	token    string
	err      error
	called   bool
	email    string
	password string
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (string, error) {
	f.called = true
	f.email = email
	f.password = password
	return f.token, f.err
}

func postLogin(svc *fakeAuthService, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handleLogin(svc)(rec, req)
	return rec
}

func TestHandleLogin(t *testing.T) {
	svc := &fakeAuthService{token: "token-de-teste"}

	rec := postLogin(svc, "application/json", `{"email":" ana@exemplo.com","password":" senha com espacos "}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200; corpo: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, esperado application/json", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, esperado no-store", got)
	}
	want := `{"access_token":"token-de-teste","token_type":"Bearer","expires_in":900}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("corpo = %s, esperado %s", got, want)
	}
	if svc.email != " ana@exemplo.com" || svc.password != " senha com espacos " {
		t.Errorf("service recebeu %q e %q; o handler nao deveria mexer nos valores", svc.email, svc.password)
	}
}

func TestHandleLoginRequisicaoRecusada(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantDetail  string
	}{
		{"text plain", "text/plain", `{"email":"ana@exemplo.com","password":"x"}`, http.StatusUnsupportedMediaType, "Content-Type deve ser application/json"},
		{"json cortado", "application/json", `{"email":`, http.StatusBadRequest, "JSON malformado"},
		{"campo desconhecido", "application/json", `{"email":"ana@exemplo.com","password":"x","role":"admin"}`, http.StatusBadRequest, `campo desconhecido: "role"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{token: "token-de-teste"}
			rec := postLogin(svc, tt.contentType, tt.body)

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

func TestHandleLoginErroDoService(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
		wantAuth   string
	}{
		{"credenciais invalidas", apperror.Unauthorized("credenciais invalidas"), http.StatusUnauthorized, "credenciais invalidas", "Bearer"},
		{"erro interno", errors.New("banco fora do ar"), http.StatusInternalServerError, "erro inesperado. tente novamente mais tarde", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{err: tt.err}
			rec := postLogin(svc, "application/json", `{"email":"ana@exemplo.com","password":"senha-errada"}`)

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
			if got := rec.Header().Get("WWW-Authenticate"); got != tt.wantAuth {
				t.Errorf("WWW-Authenticate = %q, esperado %q", got, tt.wantAuth)
			}
			if strings.Contains(rec.Body.String(), "access_token") {
				t.Errorf("resposta de erro trouxe token: %s", rec.Body.String())
			}
		})
	}
}
