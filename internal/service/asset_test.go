package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

type fakeAssetStore struct {
	assets  []domain.Asset
	err     error
	created *domain.Asset
}

func (f *fakeAssetStore) List(ctx context.Context) ([]domain.Asset, error) {
	return f.assets, f.err
}

func (f *fakeAssetStore) Create(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	f.created = &a
	a.ID = uuid.New()
	a.Version = 1
	return a, nil
}

func ptr(s string) *string {
	return &s
}

func TestAssetServiceList(t *testing.T) {
	want := []domain.Asset{
		{ID: uuid.New(), Name: "srv-web-01", Type: domain.AssetTypeServidor},
		{ID: uuid.New(), Name: "portal", Type: domain.AssetTypeAplicacao},
	}
	svc := service.NewAssetService(&fakeAssetStore{assets: want})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("recebeu %d ativos, esperado %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID {
			t.Errorf("ativo %d: ID %v, esperado %v", i, got[i].ID, want[i].ID)
		}
	}
}

func TestAssetServiceListPropagaErro(t *testing.T) {
	errBanco := errors.New("banco fora do ar")
	svc := service.NewAssetService(&fakeAssetStore{err: errBanco})

	_, err := svc.List(context.Background())
	if !errors.Is(err, errBanco) {
		t.Fatalf("erro = %v, esperado que contivesse %v", err, errBanco)
	}
}

func TestAssetServiceCreate(t *testing.T) {
	store := &fakeAssetStore{}
	svc := service.NewAssetService(store)

	in := service.CreateAssetInput{
		Name:        "  srv-web-01  ",
		Identifier:  "\t10.0.0.10\n",
		Type:        "servidor",
		Criticality: "alta",
		Description: ptr("  servidor web principal  "),
	}

	got, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if store.created == nil {
		t.Fatal("store.Create nao foi chamado")
	}

	c := store.created
	if c.Name != "srv-web-01" {
		t.Errorf("name = %q, esperado %q", c.Name, "srv-web-01")
	}
	if c.Identifier != "10.0.0.10" {
		t.Errorf("identifier = %q, esperado %q", c.Identifier, "10.0.0.10")
	}
	if c.Type != domain.AssetTypeServidor {
		t.Errorf("type = %q, esperado %q", c.Type, domain.AssetTypeServidor)
	}
	if c.Criticality != domain.CriticalityAlta {
		t.Errorf("criticality = %q, esperado %q", c.Criticality, domain.CriticalityAlta)
	}
	if c.Description == nil || *c.Description != "servidor web principal" {
		t.Errorf("description = %v, esperado %q", c.Description, "servidor web principal")
	}
	if got.ID == uuid.Nil {
		t.Error("ID vazio no ativo devolvido")
	}
}
