package service

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

const (
	maxNameLen        = 120
	maxIdentifierLen  = 255
	maxDescriptionLen = 2000
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
	a, fields := newAsset(in)
	if len(fields) > 0 {
		return domain.Asset{}, apperror.Validation(fields)
	}
	return s.store.Create(ctx, a)
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
