package domain

import (
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetTypeServidor  AssetType = "servidor"
	AssetTypeAplicacao AssetType = "aplicacao"
	AssetTypeRede      AssetType = "rede"
	AssetTypeEstacao   AssetType = "estacao"
	AssetTypeNuvem     AssetType = "nuvem"
)

func (t AssetType) Valid() bool {
	switch t {
	case AssetTypeServidor, AssetTypeAplicacao, AssetTypeRede, AssetTypeEstacao, AssetTypeNuvem:
		return true
	}
	return false
}

type Criticality string

const (
	CriticalityBaixa   Criticality = "baixa"
	CriticalityMedia   Criticality = "media"
	CriticalityAlta    Criticality = "alta"
	CriticalityCritica Criticality = "critica"
)

func (c Criticality) Valid() bool {
	switch c {
	case CriticalityBaixa, CriticalityMedia, CriticalityAlta, CriticalityCritica:
		return true
	}
	return false
}

type Asset struct {
	ID          uuid.UUID
	Name        string
	Identifier  string
	Type        AssetType
	Criticality Criticality
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	Version     int32
}
