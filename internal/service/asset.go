package service

import (
	"context"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

type assetStore interface {
	List(ctx context.Context) ([]domain.Asset, error)
	Create(ctx context.Context, a domain.Asset) (domain.Asset, error)
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

type CreateAssetInput struct {
	Name        string
	Identifier  string
	Type        string
	Criticality string
	Description *string
}

func (s *AssetService) Create(ctx context.Context, in CreateAssetInput) (domain.Asset, error) {
	a := domain.Asset{
		Name:        in.Name,
		Identifier:  in.Identifier,
		Type:        domain.AssetType(in.Type),
		Criticality: domain.Criticality(in.Criticality),
		Description: in.Description,
	}
	return s.store.Create(ctx, a)
}
