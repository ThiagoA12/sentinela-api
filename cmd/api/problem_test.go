package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

func TestWriteProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
		wantFields int
	}{
		{"interno", errors.New("pq: tabela assets não existe"), 500, "erro inesperado. tente novamente mais tarde", 0},
		{"invalido", apperror.Invalid("JSON malformado"), 400, "JSON malformado", 0},
		{"nao encontrado", apperror.NotFound("ativo não encontrado"), 404, "ativo não encontrado", 0},
		{"conflito embrulhado", fmt.Errorf("criar ativo: %w", apperror.Conflict("identificador em uso", errors.New("23505"))), 409, "identificador em uso", 0},
		{"validacao", apperror.Validation([]apperror.FieldError{{Field: "name", Reason: "obrigatório"}}), 422, "campo(s) invalidos", 1},
		{"metodo nao permitido", apperror.MethodNotAllowed("metodo nao permitido nesta rota"), 405, "metodo nao permitido nesta rota", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/ativos", nil)

			writeProblem(rec, req, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q", ct)
			}

			var p problem
			if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
				t.Fatalf("corpo não é JSON: %v", err)
			}
			if p.Type != "about:blank" || p.Status != tt.wantStatus || p.Title != http.StatusText(tt.wantStatus) {
				t.Errorf("problem = %+v", p)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if p.Instance != "/api/v1/ativos" {
				t.Errorf("instance = %q", p.Instance)
			}
			if len(p.Errors) != tt.wantFields {
				t.Errorf("errors tem %d itens, esperado %d", len(p.Errors), tt.wantFields)
			}
		})
	}
}

func TestWriteProblemNaoVazaDetalheInterno(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ativos", nil)

	writeProblem(rec, req, errors.New(`ERROR: relation "assets" does not exist (SQLSTATE 42P01)`))

	if body := rec.Body.String(); strings.Contains(body, "assets") || strings.Contains(body, "SQLSTATE") {
		t.Errorf("resposta vazou detalhe interno: %s", body)
	}
}

func TestWriteProblemRequestID(t *testing.T) {
	h := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, apperror.NotFound("ativo nao encontrado"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ativos/x", nil))

	var p problem
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("corpo nao e JSON: %v", err)
	}
	want := rec.Header().Get("X-Request-ID")
	if want == "" || p.RequestID != want {
		t.Fatalf("request_id no corpo = %q, no cabecalho = %q", p.RequestID, want)
	}
}
