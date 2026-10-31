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
