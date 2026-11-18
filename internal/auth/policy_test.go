package auth_test

import (
	"strings"
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/auth"
)

func TestValidPasswordLength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"vazia", "", false},
		{"14 caracteres", strings.Repeat("a", 14), false},
		{"15 caracteres", strings.Repeat("a", 15), true},
		{"128 caracteres", strings.Repeat("a", 128), true},
		{"129 caracteres", strings.Repeat("a", 129), false},
		{"15 letras acentuadas", strings.Repeat("ç", 15), true},
		{"14 letras acentuadas", strings.Repeat("ç", 14), false},
		{"128 letras acentuadas", strings.Repeat("ç", 128), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := auth.ValidPasswordLength(tt.password); got != tt.want {
				t.Errorf("ValidPasswordLength(%d bytes) = %v, quero %v", len(tt.password), got, tt.want)
			}
		})
	}
}
