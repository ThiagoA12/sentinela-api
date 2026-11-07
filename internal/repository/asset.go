package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/db"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AssetRepository struct {
	q *db.Queries
}

func NewAssetRepository(conn db.DBTX) *AssetRepository {
	return &AssetRepository{q: db.New(conn)}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s *string) *string {
	if s == nil {
		return nil
	}
	e := likeEscaper.Replace(*s)
	return &e
}

func (r *AssetRepository) List(ctx context.Context, f domain.AssetFilter) ([]domain.Asset, int64, error) {
	search := escapeLike(f.Search)

	total, err := r.q.CountAssets(ctx, db.CountAssetsParams{
		Type:        (*string)(f.Type),
		Criticality: (*string)(f.Criticality),
		Q:           search,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("contar ativos: %w", err)
	}

	rows, err := r.q.ListAssets(ctx, db.ListAssetsParams{
		Type:        (*string)(f.Type),
		Criticality: (*string)(f.Criticality),
		Q:           search,
		Sort:        f.Sort,
		PageSize:    int32(f.PageSize),
		PageOffset:  int32((f.Page - 1) * f.PageSize),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listar ativos: %w", err)
	}

	assets := make([]domain.Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, toDomainAsset(row))
	}
	return assets, total, nil
}

func (r *AssetRepository) Create(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	row, err := r.q.CreateAsset(ctx, db.CreateAssetParams{
		Name:        a.Name,
		Identifier:  a.Identifier,
		Type:        string(a.Type),
		Criticality: string(a.Criticality),
		Description: a.Description,
	})
	if err != nil {
		return domain.Asset{}, fmt.Errorf("criar ativo: %w", identifierInUse(err))
	}
	return toDomainAsset(row), nil
}

func (r *AssetRepository) Get(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	row, err := r.q.GetAsset(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, apperror.NotFound("ativo nao encontrado")
	}
	if err != nil {
		return domain.Asset{}, fmt.Errorf("buscar ativo: %w", err)
	}
	return toDomainAsset(row), nil
}

func (r *AssetRepository) Update(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	row, err := r.q.UpdateAsset(ctx, db.UpdateAssetParams{
		ID:          a.ID,
		Version:     a.Version,
		Name:        a.Name,
		Identifier:  a.Identifier,
		Type:        string(a.Type),
		Criticality: string(a.Criticality),
		Description: a.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, apperror.Conflict("ativo alterado por outra requisicao; busque a versao atual e tente de novo", err)
	}
	if err != nil {
		return domain.Asset{}, fmt.Errorf("atualizar ativo: %w", identifierInUse(err))
	}
	return toDomainAsset(row), nil
}

func (r *AssetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteAsset(ctx, id)
	if err != nil {
		return fmt.Errorf("excluir ativo: %w", err)
	}
	if n == 0 {
		return apperror.NotFound("ativo nao encontrado")
	}
	return nil
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
