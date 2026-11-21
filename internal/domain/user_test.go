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
