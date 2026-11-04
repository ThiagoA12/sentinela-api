package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

var fixedTime = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type fakeAssetService struct {
	err     error
	created *service.CreateAssetInput
	gotID   *uuid.UUID
	updated *service.UpdateAssetInput
}

func (f *fakeAssetService) List(ctx context.Context) ([]domain.Asset, error) {
	return nil, nil
}

func (f *fakeAssetService) Create(ctx context.Context, in service.CreateAssetInput) (domain.Asset, error) {
	f.created = &in
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	return domain.Asset{
		ID:          uuid.MustParse("019a3c1e-0000-7000-8000-000000000001"),
		Name:        in.Name,
		Identifier:  in.Identifier,
		Type:        domain.AssetType(in.Type),
		Criticality: domain.Criticality(in.Criticality),
		Description: in.Description,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
		Version:     1,
	}, nil
}

func (f *fakeAssetService) Get(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	f.gotID = &id
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	return domain.Asset{
		ID:          id,
		Name:        "Servidor web",
		Identifier:  "srv-web-01",
		Type:        domain.AssetTypeServidor,
		Criticality: domain.CriticalityAlta,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
		Version:     1,
	}, nil
}

func (f *fakeAssetService) Update(ctx context.Context, id uuid.UUID, in service.UpdateAssetInput) (domain.Asset, error) {
	f.gotID = &id
	f.updated = &in
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	return domain.Asset{
		ID:          id,
		Name:        "Servidor web",
		Identifier:  "srv-web-01",
		Type:        domain.AssetTypeServidor,
		Criticality: domain.CriticalityMedia,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
		Version:     4,
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}

func postAtivo(svc *fakeAssetService, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ativos", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handleCriaAtivo(svc)(rec, req)
	return rec
}

func getAtivo(svc *fakeAssetService, id string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ativos/{id}", handleBuscaAtivo(svc))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ativos/"+id, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCriaAtivo(t *testing.T) {
	svc := &fakeAssetService{}
	body := `{"name":"Servidor web","identifier":"srv-web-01","type":"servidor","criticality":"alta"}`

	rec := postAtivo(svc, "application/json", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado %d; corpo: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, esperado application/json", got)
	}
	wantLocation := "/api/v1/ativos/019a3c1e-0000-7000-8000-000000000001"
	if got := rec.Header().Get("Location"); got != wantLocation {
		t.Errorf("Location = %q, esperado %q", got, wantLocation)
	}

	if svc.created == nil {
		t.Fatal("service nao foi chamado")
	}
	want := service.CreateAssetInput{Name: "Servidor web", Identifier: "srv-web-01", Type: "servidor", Criticality: "alta"}
	if *svc.created != want {
		t.Errorf("entrada = %+v, esperado %+v", *svc.created, want)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta nao e JSON: %v", err)
	}
	desc, ok := resp["description"]
	if !ok {
		t.Error("chave description ausente")
	} else if desc != nil {
		t.Errorf("description = %v, esperado null", desc)
	}
	if _, ok := resp["deleted_at"]; ok {
		t.Error("deleted_at nao deveria aparecer")
	}
}

func TestCriaAtivoRequisicaoRecusada(t *testing.T) {
	valido := `{"name":"Servidor web","identifier":"srv-web-01","type":"servidor","criticality":"alta"}`
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantDetail  string
	}{
		{"sem Content-Type", "", valido, http.StatusUnsupportedMediaType, "Content-Type deve ser application/json"},
		{"text plain", "text/plain", valido, http.StatusUnsupportedMediaType, "Content-Type deve ser application/json"},
		{"content-type malformado", "application/", valido, http.StatusUnsupportedMediaType, "Content-Type deve ser application/json"},
		{"corpo acima de 1MB", "application/json", `{"name":"` + strings.Repeat("a", 1<<20) + `"}`, http.StatusRequestEntityTooLarge, "corpo maior que 1MB"},
		{"espacos depois do objeto passam de 1MB", "application/json", valido + strings.Repeat(" ", 1100000), http.StatusRequestEntityTooLarge, "corpo maior que 1MB"},
		{"corpo vazio", "application/json", "", http.StatusBadRequest, "corpo vazio"},
		{"json cortado", "application/json", `{"name":`, http.StatusBadRequest, "JSON malformado"},
		{"campo desconhecido", "application/json", `{"owner_id":"x"}`, http.StatusBadRequest, `campo desconhecido: "owner_id"`},
		{"tipo errado", "application/json", `{"name":123}`, http.StatusBadRequest, "campo name com tipo invalido"},
		{"array no lugar de objeto", "application/json", `[1]`, http.StatusBadRequest, "o corpo deve ser um objeto JSON"},
		{"dois objetos", "application/json", valido + valido, http.StatusBadRequest, "o corpo deve ter um unico objeto json"},
		{"lixo depois do objeto", "application/json", valido + " x", http.StatusBadRequest, "o corpo deve ter um unico objeto json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAssetService{}
			rec := postAtivo(svc, tt.contentType, tt.body)

			var p struct {
				Status int    `json:"status"`
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e JSON: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d; corpo: %s", rec.Code, tt.wantStatus, p.Detail)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, esperado application/problem+json", got)
			}

			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}

			if svc.created != nil {
				t.Errorf("service nao deveria ter sido chamado")
			}
		})
	}
}

func TestCriaAtivoErroDoService(t *testing.T) {
	valido := `{"name":"Servidor web", "identifier":"srv-web-01", "type":"servidor", "criticality":"alta"}`

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
		wantField  int
		naoVazar   string
	}{
		{
			name: "validacao",
			err: apperror.Validation([]apperror.FieldError{
				{Field: "name", Reason: "obrigatorio"},
				{Field: "type", Reason: "invalido"},
			}),
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "campo(s) invalidos",
			wantField:  2,
		},
		{
			name:       "conflito",
			err:        apperror.Conflict("ja existe um ativo com este identificador", errors.New("duplicate key value violates unique constraint")),
			wantStatus: http.StatusConflict,
			wantDetail: "ja existe um ativo com este identificador",
			naoVazar:   "duplicate key",
		},
		{
			name:       "erro interno",
			err:        errors.New(`pq: relation "assets" does not exist`),
			wantStatus: http.StatusInternalServerError,
			wantDetail: "erro inesperado. tente novamente mais tarde",
			naoVazar:   "relation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAssetService{err: tt.err}
			rec := postAtivo(svc, "application/json", valido)

			var p struct {
				Detail string `json:"detail"`
				Errors []struct {
					Field  string `json:"field"`
					Reason string `json:"reason"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e json: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, esperado application/problem+json", got)
			}
			if got := rec.Header().Get("Location"); got != "" {
				t.Errorf("Location = %q, nao deveria existir em erro", got)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if len(p.Errors) != tt.wantField {
				t.Errorf("errors tem %d itens, esperado %d", len(p.Errors), tt.wantField)
			}
			if tt.naoVazar != "" && strings.Contains(rec.Body.String(), tt.naoVazar) {
				t.Errorf("resposta vazou %q: %s", tt.naoVazar, rec.Body.String())
			}
			if svc.created == nil {
				t.Error("service deveria ter sido chamado")
			}
		})
	}
}

func TestBuscaAtivo(t *testing.T) {
	id := uuid.MustParse("019a3c1e-0000-7000-8000-000000000001")
	svc := &fakeAssetService{}
	rec := getAtivo(svc, id.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200; corpo: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, esperado application/json", got)
	}
	if svc.gotID == nil {
		t.Fatal("service nao foi chamado")
	}
	if *svc.gotID != id {
		t.Errorf("service recebeu %s, esperado %s", *svc.gotID, id)
	}

	var body ativoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta nao e JSON: %v", err)
	}
	if body.ID != id || body.Name != "Servidor web" || body.Version != 1 {
		t.Errorf("corpo inesperado: %+v", body)
	}
}

func TestBuscaAtivoErros(t *testing.T) {
	valido := "019a3c1e-0000-7000-8000-000000000001"
	tests := []struct {
		name        string
		id          string
		err         error
		wantStatus  int
		wantDetail  string
		wantChamado bool
	}{
		{"id invalido", "abc", nil, http.StatusBadRequest, "id invalido", false},
		{"quase uuid", "019a3c1e-0000-7000-8000-00000000000g", nil, http.StatusBadRequest, "id invalido", false},
		{"nao encontrado", valido, apperror.NotFound("ativo nao encontrado"), http.StatusNotFound, "ativo nao encontrado", true},
		{"erro interno", valido, errors.New("conexao recusada"), http.StatusInternalServerError, "erro inesperado. tente novamente mais tarde", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAssetService{err: tt.err}
			rec := getAtivo(svc, tt.id)

			var p struct {
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e JSON: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, esperado application/problem+json", got)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if chamado := svc.gotID != nil; chamado != tt.wantChamado {
				t.Errorf("service chamado = %v, esperado %v", chamado, tt.wantChamado)
			}
		})
	}
}

func patchAtivo(svc *fakeAssetService, id, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/ativos/{id}", handleAtualizaAtivo(svc))
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/ativos/"+id, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAtualizaAtivo(t *testing.T) {
	id := uuid.MustParse("019a3c1e-0000-7000-8000-000000000001")
	svc := &fakeAssetService{}
	rec := patchAtivo(svc, id.String(), `{"criticality":"media","description":"","version":3}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200; corpo: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, esperado application/json", got)
	}
	if svc.updated == nil {
		t.Fatal("service nao foi chamado")
	}
	if *svc.gotID != id {
		t.Errorf("service recebeu id %s, esperado %s", *svc.gotID, id)
	}

	want := service.UpdateAssetInput{
		Criticality: ptr("media"),
		Description: ptr(""),
		Version:     ptr(int32(3)),
	}
	if !reflect.DeepEqual(*svc.updated, want) {
		t.Errorf("service recebeu %+v, esperado %+v", *svc.updated, want)
	}

	var body ativoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta nao e JSON: %v", err)
	}
	if body.Version != 4 {
		t.Errorf("version = %d, esperado 4", body.Version)
	}
}

func TestAtualizaAtivoErros(t *testing.T) {
	valido := "019a3c1e-0000-7000-8000-000000000001"
	tests := []struct {
		name        string
		id          string
		body        string
		err         error
		wantStatus  int
		wantDetail  string
		wantChamado bool
	}{
		{"id invalido", "abc", `{"version":3}`, nil, http.StatusBadRequest, "id invalido", false},
		{"version como texto", valido, `{"version":"3"}`, nil, http.StatusBadRequest, "campo version com tipo invalido", false},
		{"tenta trocar o id", valido, `{"id":"019a3c1e-0000-7000-8000-000000000002","version":3}`, nil, http.StatusBadRequest, `campo desconhecido: "id"`, false},
		{"validacao", valido, `{"name":"  ","version":3}`, apperror.Validation([]apperror.FieldError{{Field: "name", Reason: "obrigatorio"}}), http.StatusUnprocessableEntity, "campo(s) invalidos", true},
		{"nao encontrado", valido, `{"version":3}`, apperror.NotFound("ativo nao encontrado"), http.StatusNotFound, "ativo nao encontrado", true},
		{"versao desatualizada", valido, `{"name":"Novo","version":2}`, apperror.Conflict("ativo alterado por outra requisicao; busque a versao atual e tente de novo", nil), http.StatusConflict, "ativo alterado por outra requisicao; busque a versao atual e tente de novo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAssetService{err: tt.err}
			rec := patchAtivo(svc, tt.id, tt.body)

			var p struct {
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("resposta nao e JSON: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, esperado application/problem+json", got)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, esperado %q", p.Detail, tt.wantDetail)
			}
			if chamado := svc.updated != nil; chamado != tt.wantChamado {
				t.Errorf("service chamado = %v, esperado %v", chamado, tt.wantChamado)
			}
		})
	}
}
