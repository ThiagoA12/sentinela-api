package repository

import (
	"context"
	"fmt"

	"github.com/ThiagoA12/sentinela-api/internal/db"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

type AssetRepository struct {
	q *db.Queries
}

func NewAssetRepository(conn db.DBTX) *AssetRepository {
	return &AssetRepository{q: db.New(conn)}
}

func (r *AssetRepository) List(ctx context.Context) ([]domain.Asset, error) {
	rows, err := r.q.ListAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar ativos: %w", err)
	}

	assets := make([]domain.Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, toDomainAsset(row))
	}
	return assets, nil
}

func toDomainAsset(a db.Asset) domain.Asset {
	return domain.Asset{
		ID:          a.ID,
		Name:        a.Name,
		Identifier:  a.Identifier,
		Type:        domain.AssetType(a.Type),
		Criticality: domain.Criticality(a.Criticality),
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		DeletedAt:   a.DeletedAt,
		Version:     a.Version,
	}
}
