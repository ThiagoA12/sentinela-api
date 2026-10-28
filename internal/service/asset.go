package service

import (
	"context"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

type assetStore interface {
	List(ctx context.Context) ([]domain.Asset, error)
}

type AssetService struct {
	store assetStore
}

func NewAssetService(store assetStore) *AssetService {
	return &AssetService{store: store}
}
func (s *AssetService) List(ctx context.Context) ([]domain.Asset, error) {
	return s.store.List(ctx)
}
