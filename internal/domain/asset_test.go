package domain_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

func TestAssetTypeValid(t *testing.T) {
	tests := []struct {
		name string
		in   domain.AssetType
		want bool
	}{
		{"servidor", domain.AssetTypeServidor, true},
		{"aplicacao", domain.AssetTypeAplicacao, true},
		{"rede", domain.AssetTypeRede, true},
		{"estacao", domain.AssetTypeEstacao, true},
		{"nuvem", domain.AssetTypeNuvem, true},
		{"vazio", "", false},
		{"desconhecido", "banana", false},
		{"maiuscula", "Servidor", false},
		{"espaco nas pontas", " servidor ", false},
		{"com acento", "aplicação", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Valid(); got != tt.want {
				t.Errorf("AssetType(%q).Valid() = %v, esperado %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCriticalityValid(t *testing.T) {
	tests := []struct {
		name string
		in   domain.Criticality
		want bool
	}{
		{"baixa", domain.CriticalityBaixa, true},
		{"media", domain.CriticalityMedia, true},
		{"alta", domain.CriticalityAlta, true},
		{"critica", domain.CriticalityCritica, true},
		{"vazio", "", false},
		{"desconhecido", "urgente", false},
		{"maiuscula", "CRITICA", false},
		{"com acento", "média", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Valid(); got != tt.want {
				t.Errorf("Criticality(%q).Valid() = %v, esperado %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAssetEditableBy(t *testing.T) {
	dono := uuid.MustParse("01a1cab1-9c80-7a10-8000-000000000001")
	outro := uuid.MustParse("01a1cab1-9c80-7a10-8000-000000000002")
	ativo := domain.Asset{OwnerID: dono}

	tests := []struct {
		name string
		u    domain.User
		want bool
	}{
		{"admin responsavel", domain.User{ID: dono, Role: domain.RoleAdmin}, true},
		{"admin de ativo alheio", domain.User{ID: outro, Role: domain.RoleAdmin}, true},
		{"analista responsavel", domain.User{ID: dono, Role: domain.RoleAnalista}, true},
		{"analista de ativo alheio", domain.User{ID: outro, Role: domain.RoleAnalista}, false},
		{"leitor responsavel", domain.User{ID: dono, Role: domain.RoleLeitor}, false},
		{"leitor de ativo alheio", domain.User{ID: outro, Role: domain.RoleLeitor}, false},
		{"papel desconhecido responsavel", domain.User{ID: dono, Role: "root"}, false},
		{"sem papel responsavel", domain.User{ID: dono}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ativo.EditableBy(tt.u); got != tt.want {
				t.Errorf("EditableBy = %v, esperado %v", got, tt.want)
			}
		})
	}
}
