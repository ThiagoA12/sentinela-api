package auth_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/auth"
)

const senha = "cavalo correto bateria grampo"

func TestHashPasswordFormato(t *testing.T) {
	encoded := auth.HashPassword(senha)

	const prefix = "$argon2id$v=19$m=19456,t=2,p=1$"
	if !strings.HasPrefix(encoded, prefix) {
		t.Fatalf("hash = %q, quero comecando com %q", encoded, prefix)
	}
	parts := strings.Split(strings.TrimPrefix(encoded, prefix), "$")
	if len(parts) != 2 {
		t.Fatalf("hash = %q, quero salt e chave depois dos parametros", encoded)
	}
	if salt, err := base64.RawStdEncoding.DecodeString(parts[0]); err != nil || len(salt) != 16 {
		t.Errorf("salt = %q (%v), quero 16 bytes em base64", parts[0], err)
	}
	if key, err := base64.RawStdEncoding.DecodeString(parts[1]); err != nil || len(key) != 32 {
		t.Errorf("chave = %q (%v), quero 32 bytes em base64", parts[1], err)
	}
	if strings.Contains(encoded, senha) {
		t.Errorf("o hash contem a senha em texto")
	}
}

func TestHashPasswordSaltNovoACadaVez(t *testing.T) {
	a := auth.HashPassword(senha)
	b := auth.HashPassword(senha)
	if a == b {
		t.Errorf("dois hashes da mesma senha sairam iguais: o salt nao esta mudando\n%s", a)
	}
}

func TestVerifyPassword(t *testing.T) {
	encoded := auth.HashPassword(senha)
	parts := strings.Split(encoded, "$")
	saltCorrompido := strings.Join([]string{parts[0], parts[1], parts[2], parts[3], "!!!!", parts[5]}, "$")

	tests := []struct {
		name     string
		password string
		encoded  string
		want     bool
		wantErr  error
	}{
		{"senha certa", senha, encoded, true, nil},
		{"senha errada", senha + "s", encoded, false, nil},
		{"senha vazia", "", encoded, false, nil},
		{"conta travada", senha, "!", false, auth.ErrInvalidHash},
		{"outro algoritmo", senha, strings.Replace(encoded, "argon2id", "argon2i", 1), false, auth.ErrInvalidHash},
		{"parametros alterados", senha, strings.Replace(encoded, "m=19456", "m=4194304", 1), false, auth.ErrInvalidHash},
		{"salt corrompido", senha, saltCorrompido, false, auth.ErrInvalidHash},
		{"chave cortada", senha, encoded[:len(encoded)-4], false, auth.ErrInvalidHash},
		{"sem a chave", senha, strings.Join(parts[:5], "$"), false, auth.ErrInvalidHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.VerifyPassword(tt.password, tt.encoded)
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("VerifyPassword = %v, %v; quero %v, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
