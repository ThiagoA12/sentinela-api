package domain_test

import (
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

func TestRoleValid(t *testing.T) {
	tests := []struct {
		name string
		in   domain.Role
		want bool
	}{
		{"admin", domain.RoleAdmin, true},
		{"analista", domain.RoleAnalista, true},
		{"leitor", domain.RoleLeitor, true},
		{"vazio", "", false},
		{"desconhecido", "root", false},
		{"maiuscula", "Admin", false},
		{"espaco nas pontas", " admin ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Valid(); got != tt.want {
				t.Errorf("Role(%q).Valid() = %v, esperado %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestUserCanOwnAssets(t *testing.T) {
	tests := []struct {
		name string
		u    domain.User
		want bool
	}{
		{"admin ativo", domain.User{Role: domain.RoleAdmin, Active: true}, true},
		{"analista ativo", domain.User{Role: domain.RoleAnalista, Active: true}, true},
		{"leitor ativo", domain.User{Role: domain.RoleLeitor, Active: true}, false},
		{"admin desativado", domain.User{Role: domain.RoleAdmin}, false},
		{"analista desativado", domain.User{Role: domain.RoleAnalista}, false},
		{"papel desconhecido", domain.User{Role: "root", Active: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.CanOwnAssets(); got != tt.want {
				t.Errorf("CanOwnAssets = %v, esperado %v", got, tt.want)
			}
		})
	}
}
