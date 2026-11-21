package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

func TestIdentifierInUse(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want apperror.Kind
	}{
		{"indice do identificador", &pgconn.PgError{Code: "23505", ConstraintName: "assets_identifier_active_key"}, apperror.KindConflict},
		{"embrulhado", fmt.Errorf("criar ativo: %w", &pgconn.PgError{Code: "23505", ConstraintName: "assets_identifier_active_key"}), apperror.KindConflict},
		{"outro indice unico", &pgconn.PgError{Code: "23505", ConstraintName: "assets_pkey"}, apperror.KindInternal},
		{"outro codigo", &pgconn.PgError{Code: "23514", ConstraintName: "assets_type_check"}, apperror.KindInternal},
		{"erro comum", errors.New("conexão recusada"), apperror.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apperror.KindOf(identifierInUse(tt.err)); got != tt.want {
				t.Errorf("Kind = %v, esperado %v", got, tt.want)
			}
		})
	}
}

func TestEmailInUse(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want apperror.Kind
	}{
		{"indice do e-mail", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_lower_key"}, apperror.KindConflict},
		{"embrulhado", fmt.Errorf("criar usuario: %w", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_lower_key"}), apperror.KindConflict},
		{"indice de ativos", &pgconn.PgError{Code: "23505", ConstraintName: "assets_identifier_active_key"}, apperror.KindInternal},
		{"outro codigo", &pgconn.PgError{Code: "23514", ConstraintName: "users_role_check"}, apperror.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apperror.KindOf(emailInUse(tt.err)); got != tt.want {
				t.Errorf("Kind = %v, esperado %v", got, tt.want)
			}
		})
	}
}
