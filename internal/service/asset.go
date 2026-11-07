package service

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/google/uuid"
)

const (
	maxNameLen        = 120
	maxIdentifierLen  = 255
	maxDescriptionLen = 2000
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	defaultSort     = "created_at"
	maxPage         = 10000
	maxPageSize     = 100
	maxSearchLen    = 100
)

const (
	reasonType        = "deve ser: servidor, aplicacao, rede, estacao ou nuvem"
	reasonCriticality = "deve ser: baixa, media, alta ou critica"
)

var assetSortFields = []string{"name", "identifier", "type", "criticality", "created_at", "updated_at"}

type assetStore interface {
	List(ctx context.Context, f domain.AssetFilter) ([]domain.Asset, int64, error)
	Create(ctx context.Context, a domain.Asset) (domain.Asset, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Asset, error)
	Update(ctx context.Context, a domain.Asset) (domain.Asset, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type AssetService struct {
	store assetStore
}

func NewAssetService(store assetStore) *AssetService {
	return &AssetService{store: store}
}

type ListAssetsInput struct {
	Type        *string
	Criticality *string
	Search      *string
	Sort        *string
	Page        *int
	PageSize    *int
}

type AssetPage struct {
	Assets   []domain.Asset
	Page     int
	PageSize int
	Total    int64
}

func (s *AssetService) List(ctx context.Context, in ListAssetsInput) (AssetPage, error) {
	f, fields := newAssetFilter(in)
	if len(fields) > 0 {
		return AssetPage{}, apperror.Validation(fields)
	}

	assets, total, err := s.store.List(ctx, f)
	if err != nil {
		return AssetPage{}, err
	}
	return AssetPage{Assets: assets, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

func newAssetFilter(in ListAssetsInput) (domain.AssetFilter, []apperror.FieldError) {
	var fields []apperror.FieldError
	f := domain.AssetFilter{Page: defaultPage, PageSize: defaultPageSize, Sort: defaultSort}

	if in.Page != nil {
		if *in.Page < 1 || *in.Page > maxPage {
			fields = append(fields, outOfRange("page", 1, maxPage))
		}
		f.Page = *in.Page
	}

	if in.PageSize != nil {
		if *in.PageSize < 1 || *in.PageSize > maxPageSize {
			fields = append(fields, outOfRange("page_size", 1, maxPageSize))
		}
		f.PageSize = *in.PageSize
	}

	if in.Type != nil {
		t := domain.AssetType(*in.Type)
		if !t.Valid() {
			fields = append(fields, apperror.FieldError{Field: "type", Reason: reasonType})
		}
		f.Type = &t
	}

	if in.Criticality != nil {
		c := domain.Criticality(*in.Criticality)
		if !c.Valid() {
			fields = append(fields, apperror.FieldError{Field: "criticality", Reason: reasonCriticality})
		}
		f.Criticality = &c
	}

	if in.Search != nil {
		switch q := strings.TrimSpace(*in.Search); {
		case utf8.RuneCountInString(q) > maxSearchLen:
			fields = append(fields, tooLong("q", maxSearchLen))
		case q != "":
			f.Search = &q
		}
	}

	if in.Sort != nil {
		if !slices.Contains(assetSortFields, strings.TrimPrefix(*in.Sort, "-")) {
			fields = append(fields, apperror.FieldError{
				Field:  "sort",
				Reason: "deve ser um de: " + strings.Join(assetSortFields, ", ") + "; prefixo - para decrescente",
			})
		}
		f.Sort = *in.Sort
	}

	return f, fields
}

type CreateAssetInput struct {
	Name        string
	Identifier  string
	Type        string
	Criticality string
	Description *string
}

func (s *AssetService) Create(ctx context.Context, in CreateAssetInput) (domain.Asset, error) {
	a, fields := newAsset(in)
	if len(fields) > 0 {
		return domain.Asset{}, apperror.Validation(fields)
	}
	return s.store.Create(ctx, a)
}

func (s *AssetService) Get(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	return s.store.Get(ctx, id)
}

type UpdateAssetInput struct {
	Name        *string
	Identifier  *string
	Type        *string
	Criticality *string
	Description *string
	Version     *int32
}

func (s *AssetService) Update(ctx context.Context, id uuid.UUID, in UpdateAssetInput) (domain.Asset, error) {
	if in.Version == nil {
		return domain.Asset{}, apperror.Validation([]apperror.FieldError{{Field: "version", Reason: "obrigatorio"}})
	}

	current, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Asset{}, err
	}

	a, fields := newAsset(mergeAsset(current, in))
	if len(fields) > 0 {
		return domain.Asset{}, apperror.Validation(fields)
	}
	a.ID = id
	a.Version = *in.Version
	return s.store.Update(ctx, a)
}

func (s *AssetService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}

func mergeAsset(current domain.Asset, in UpdateAssetInput) CreateAssetInput {
	merged := CreateAssetInput{
		Name:        current.Name,
		Identifier:  current.Identifier,
		Type:        string(current.Type),
		Criticality: string(current.Criticality),
		Description: current.Description,
	}

	if in.Name != nil {
		merged.Name = *in.Name
	}
	if in.Identifier != nil {
		merged.Identifier = *in.Identifier
	}
	if in.Type != nil {
		merged.Type = *in.Type
	}
	if in.Criticality != nil {
		merged.Criticality = *in.Criticality
	}
	if in.Description != nil {
		merged.Description = in.Description
	}
	return merged
}

func newAsset(in CreateAssetInput) (domain.Asset, []apperror.FieldError) {
	var fields []apperror.FieldError

	name := strings.TrimSpace(in.Name)
	fields = checkText(fields, "name", name, maxNameLen)

	identifier := strings.TrimSpace(in.Identifier)
	fields = checkText(fields, "identifier", identifier, maxIdentifierLen)

	assetType := domain.AssetType(in.Type)
	if !assetType.Valid() {
		fields = append(fields, apperror.FieldError{
			Field:  "type",
			Reason: reasonType,
		})
	}

	criticality := domain.Criticality(in.Criticality)
	if !criticality.Valid() {
		fields = append(fields, apperror.FieldError{
			Field:  "criticality",
			Reason: reasonCriticality,
		})
	}

	var description *string
	if in.Description != nil {
		switch d := strings.TrimSpace(*in.Description); {
		case utf8.RuneCountInString(d) > maxDescriptionLen:
			fields = append(fields, tooLong("description", maxDescriptionLen))
		case d != "":
			description = &d
		}
	}

	return domain.Asset{
		Name:        name,
		Identifier:  identifier,
		Type:        assetType,
		Criticality: criticality,
		Description: description,
	}, fields
}

func checkText(fields []apperror.FieldError, field, value string, limit int) []apperror.FieldError {
	switch {
	case value == "":
		return append(fields, apperror.FieldError{Field: field, Reason: "obrigatorio"})
	case utf8.RuneCountInString(value) > limit:
		return append(fields, tooLong(field, limit))
	}
	return fields
}

func tooLong(field string, limit int) apperror.FieldError {
	return apperror.FieldError{
		Field:  field,
		Reason: "maximo de " + strconv.Itoa(limit) + " caracteres",
	}
}

func outOfRange(field string, lo, hi int) apperror.FieldError {
	return apperror.FieldError{
		Field:  field,
		Reason: "deve estar entre " + strconv.Itoa(lo) + " e " + strconv.Itoa(hi),
	}
}
