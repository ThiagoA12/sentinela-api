package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/auth"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

func TestWithRequestID(t *testing.T) {
	var seen string
	h := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "vindo-do-cliente")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-ID")
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("X-Request-ID = %q, quer um UUID", got)
	}
	if seen != got {
		t.Fatalf("id no contexto = %q, no cabecalho = %q", seen, got)
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec2.Header().Get("X-Request-ID") == got {
		t.Fatalf("duas requisicoes com o mesmo id %q", got)
	}
}

func TestRequestIDFromSemID(t *testing.T) {
	if got := requestIDFrom(context.Background()); got != "" {
		t.Fatalf("requestIDFrom = %q, quer vazio", got)
	}
}

func TestWithRecover(t *testing.T) {
	h := withRequestID(withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("segredo interno")
	})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ativos", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, quer 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	if strings.Contains(body, "segredo") || strings.Contains(body, "goroutine") {
		t.Fatalf("resposta vazou detalhe interno: %s", body)
	}
	var p problem
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("corpo nao e JSON: %v", err)
	}
	if want := rec.Header().Get("X-Request-ID"); p.RequestID != want {
		t.Fatalf("request_id no corpo = %q, no cabecalho = %q", p.RequestID, want)
	}
}

func TestWithRecoverSemPanic(t *testing.T) {
	h := withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, quer 204", rec.Code)
	}
}

func TestWithRecoverRepassaErrAbortHandler(t *testing.T) {
	h := withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Fatalf("recover = %v, quer http.ErrAbortHandler", v)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
}

type accessEntry struct {
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Method    string `json:"method"`
	Route     string `json:"route"`
	Status    int    `json:"status"`
	Latency   *int64 `json:"latency"`
}

func registraAcesso(t *testing.T, target string, handler http.HandlerFunc) (accessEntry, *httptest.ResponseRecorder, string) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ativos/{id}", handler)
	h := withRequestID(withAccessLog(logger, withRecover(mux)))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	var e accessEntry
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("log nao e JSON: %v (%s)", err, buf.String())
	}
	return e, rec, buf.String()
}

func TestWithAccessLog(t *testing.T) {
	e, rec, raw := registraAcesso(t, "/api/v1/ativos/id-secreto?senha=x", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	if e.Msg != "requisicao" || e.Method != http.MethodGet || e.Route != "GET /api/v1/ativos/{id}" {
		t.Fatalf("entrada = %+v", e)
	}
	if e.Status != http.StatusNotFound {
		t.Fatalf("status no log = %d, quer 404", e.Status)
	}
	if want := rec.Header().Get("X-Request-ID"); e.RequestID != want {
		t.Fatalf("request_id no log = %q, no cabecalho = %q", e.RequestID, want)
	}
	if e.Latency == nil {
		t.Fatalf("log sem latency: %s", raw)
	}
	if strings.Contains(raw, "senha") || strings.Contains(raw, "secreto") {
		t.Fatalf("log com dado da URL: %s", raw)
	}
}

func TestWithAccessLogStatusPadrao(t *testing.T) {
	e, _, _ := registraAcesso(t, "/api/v1/ativos/123", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	if e.Status != http.StatusOK {
		t.Fatalf("status no log = %d, quer 200", e.Status)
	}
}

func TestWithAccessLogPanic(t *testing.T) {
	e, _, _ := registraAcesso(t, "/api/v1/ativos/123", func(w http.ResponseWriter, r *http.Request) {
		panic("falhou")
	})

	if e.Status != http.StatusInternalServerError {
		t.Fatalf("status no log = %d, quer 500", e.Status)
	}
}

func rotasDeTeste() http.Handler {
	mux := http.NewServeMux()
	ok := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
	mux.HandleFunc("GET /api/v1/ativos", ok)
	mux.HandleFunc("POST /api/v1/ativos", ok)
	mux.HandleFunc("GET /api/v1/ativos/{id}", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.PathValue("id"))
	})
	return withRouteProblems(mux)
}

func TestWithRouteProblems(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantAllow  string
		wantDetail string
	}{
		{"rota inexistente", http.MethodGet, "/nao-existe", 404, "", "rota nao encontrada"},
		{"barra no fim", http.MethodGet, "/api/v1/ativos/", 404, "", "rota nao encontrada"},
		{"metodo errado na colecao", http.MethodPut, "/api/v1/ativos", 405, "GET, HEAD, POST", "metodo nao permitido nesta rota"},
		{"metodo errado no item", http.MethodDelete, "/api/v1/ativos/abc", 405, "GET, HEAD", "metodo nao permitido nesta rota"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rotasDeTeste().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quer %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, quer %q", got, tt.wantAllow)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Fatalf("Content-Type = %q", ct)
			}
			var p problem
			if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
				t.Fatalf("corpo nao e JSON: %v", err)
			}
			if p.Detail != tt.wantDetail {
				t.Fatalf("detail = %q, quer %q", p.Detail, tt.wantDetail)
			}
		})
	}
}

func TestWithRouteProblemsRotaExistente(t *testing.T) {
	rec := httptest.NewRecorder()
	rotasDeTeste().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ativos/abc", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "abc" {
		t.Fatalf("status = %d, corpo = %q; quer 200 e \"abc\"", rec.Code, rec.Body.String())
	}
}

type fakeAuthenticator struct {
	user     domain.User
	err      error
	called   bool
	gotToken string
}

func (f *fakeAuthenticator) Authenticate(ctx context.Context, token string) (domain.User, error) {
	f.called = true
	f.gotToken = token
	return f.user, f.err
}

var (
	errTokenDeTeste = apperror.Unauthorized("token ausente, invalido ou expirado")
	writerRoles     = []domain.Role{domain.RoleAdmin, domain.RoleAnalista}
)

type authResult struct {
	rec    *httptest.ResponseRecorder
	log    string
	called bool
	user   domain.User
}

func autentica(t *testing.T, authn *fakeAuthenticator, roles []domain.Role, authorization string) authResult {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	var res authResult
	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/ativos", withAuth(authn, logger, roles, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res.called = true
		res.user, _ = userFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ativos", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res.rec = httptest.NewRecorder()
	withRequestID(mux).ServeHTTP(res.rec, req)
	res.log = buf.String()
	return res
}

func TestWithAuth(t *testing.T) {
	ana := domain.User{ID: uuid.New(), Name: "Ana", Role: domain.RoleAnalista, Active: true}

	tests := []struct {
		name          string
		authorization string
		wantToken     string
	}{
		{"bearer", "Bearer abc.def.ghi", "abc.def.ghi"},
		{"bearer em minusculas", "bearer abc.def.ghi", "abc.def.ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authn := &fakeAuthenticator{user: ana}
			res := autentica(t, authn, writerRoles, tt.authorization)

			if res.rec.Code != http.StatusNoContent || !res.called {
				t.Fatalf("status = %d, handler chamado = %v; quero 204 e chamado", res.rec.Code, res.called)
			}
			if authn.gotToken != tt.wantToken {
				t.Errorf("token = %q, quero %q", authn.gotToken, tt.wantToken)
			}
			if res.user != ana {
				t.Errorf("usuario no context = %+v, quero %+v", res.user, ana)
			}
			if res.log != "" {
				t.Errorf("log inesperado: %s", res.log)
			}
		})
	}
}

func TestWithAuthRecusa(t *testing.T) {
	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin, Active: true}
	leitor := domain.User{ID: uuid.New(), Role: domain.RoleLeitor, Active: true}

	tests := []struct {
		name          string
		authorization string
		authn         *fakeAuthenticator
		roles         []domain.Role
		wantToken     string
		wantStatus    int
		wantSecurity  bool
	}{
		{"sem cabecalho", "", &fakeAuthenticator{err: errTokenDeTeste}, writerRoles, "", 401, false},
		{"outro esquema", "Basic YW5hOnNlbmhh", &fakeAuthenticator{err: errTokenDeTeste}, writerRoles, "", 401, false},
		{"bearer sem token", "Bearer ", &fakeAuthenticator{err: errTokenDeTeste}, writerRoles, "", 401, false},
		{"token forjado", "Bearer forjado", &fakeAuthenticator{err: fmt.Errorf("%w: %w", errTokenDeTeste, auth.ErrInvalidToken)}, writerRoles, "forjado", 401, true},
		{"token expirado", "Bearer vencido", &fakeAuthenticator{err: fmt.Errorf("%w: %w", errTokenDeTeste, auth.ErrExpiredToken)}, writerRoles, "vencido", 401, false},
		{"papel sem permissao", "Bearer bom", &fakeAuthenticator{user: leitor}, writerRoles, "bom", 403, false},
		{"rota sem papel", "Bearer bom", &fakeAuthenticator{user: admin}, nil, "bom", 403, false},
		{"erro do banco", "Bearer bom", &fakeAuthenticator{err: errors.New("banco fora do ar")}, writerRoles, "bom", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := autentica(t, tt.authn, tt.roles, tt.authorization)

			if res.rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d; corpo: %s", res.rec.Code, tt.wantStatus, res.rec.Body.String())
			}
			if res.called {
				t.Error("handler chamado numa recusa")
			}
			if tt.authn.gotToken != tt.wantToken {
				t.Errorf("token = %q, quero %q", tt.authn.gotToken, tt.wantToken)
			}
			if strings.Contains(res.log, `"security":true`) != tt.wantSecurity {
				t.Errorf("evento de seguranca no log = %v, quero %v: %s", !tt.wantSecurity, tt.wantSecurity, res.log)
			}
			if tt.wantToken != "" && strings.Contains(res.log, tt.wantToken) {
				t.Errorf("log com o token: %s", res.log)
			}
		})
	}
}

type securityEntry struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Security  bool   `json:"security"`
	RequestID string `json:"request_id"`
	Route     string `json:"route"`
}

func TestWithAuthEventoDeSeguranca(t *testing.T) {
	authn := &fakeAuthenticator{err: fmt.Errorf("%w: %w", errTokenDeTeste, auth.ErrInvalidToken)}
	res := autentica(t, authn, writerRoles, "Bearer forjado")

	var e securityEntry
	if err := json.Unmarshal([]byte(res.log), &e); err != nil {
		t.Fatalf("log nao e uma linha JSON: %v (%s)", err, res.log)
	}
	want := securityEntry{
		Level:     "WARN",
		Msg:       "token invalido",
		Security:  true,
		RequestID: res.rec.Header().Get("X-Request-ID"),
		Route:     "POST /api/v1/ativos",
	}
	if e != want {
		t.Errorf("log = %+v\nquero %+v", e, want)
	}
}
