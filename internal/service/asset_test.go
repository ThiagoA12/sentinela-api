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
	assets []domain.Asset
	err    error
}

func (f *fakeAssetStore) List(ctx context.Context) ([]domain.Asset, error) {
	return f.assets, f.err
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
