package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
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
