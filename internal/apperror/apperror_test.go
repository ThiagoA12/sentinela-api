package apperror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

func TestKindOf(t *testing.T) {
	causa := errors.New("violacao de indice")

	tests := []struct {
		name string
		err  error
		want apperror.Kind
	}{
		{"erro comum", errors.New("falhou"), apperror.KindInternal},
		{"nil", nil, apperror.KindInternal},
		{"not founnd", apperror.NotFound("ativo nao encontrado"), apperror.KindNotFound},
		{"conflito embrulhado", fmt.Errorf("criar ativo: %w", apperror.Conflict("duplicado", causa)), apperror.KindConflict},
		{"embrulhado duas vezes", fmt.Errorf("a: %w", fmt.Errorf("b: %w", apperror.Invalid("x"))), apperror.KindInvalid},
		{"sem Kind", &apperror.Error{Detail: "esquecido"}, apperror.KindInternal},
		{"corpo grande", apperror.TooLarge("corpo maior que 1 MB"), apperror.KindTooLarge},
		{"midia nao suportada", apperror.UnsupportedMedia("Content-Type deve ser application/json"), apperror.KindUnsupportedMedia},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apperror.KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf(%v) = %v, esperado %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorPreservaCausa(t *testing.T) {
	causa := errors.New("violação de índice")
	err := fmt.Errorf("criar ativo: %w", apperror.Conflict("identificador em uso", causa))

	if !errors.Is(err, causa) {
		t.Fatal("errors.Is não encontrou a causa original")
	}
	if got, want := err.Error(), "criar ativo: identificador em uso: violação de índice"; got != want {
		t.Errorf("Error() = %q, esperado %q", got, want)
	}
}
