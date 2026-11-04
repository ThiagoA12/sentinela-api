package service

import (
	"context"
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

type assetStore interface {
	List(ctx context.Context) ([]domain.Asset, error)
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
			Reason: "deve ser: servidor, aplicacao, rede, estacao ou nuvem",
		})
	}

	criticality := domain.Criticality(in.Criticality)
	if !criticality.Valid() {
		fields = append(fields, apperror.FieldError{
			Field:  "criticality",
			Reason: "deve ser: baixa, media, alta ou critica",
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
