package domain_test

import (
	"testing"

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
