package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

var fixedTime = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type fakeAssetService struct {
	err     error
	created *service.CreateAssetInput
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

func postAtivo(svc *fakeAssetService, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ativos", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handleCriaAtivo(svc)(rec, req)
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
