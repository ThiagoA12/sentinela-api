package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ativo. Como ele sai na resposta da api.
type Ativo struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Identifier  string    `json:"identifier" db:"identifier"`
	Type        string    `json:"type" db:"type"`
	Criticality string    `json:"criticality" db:"criticality"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// listaAtivos, corpo da resposta. a lista vai dentro de "data".
type listaAtivos struct {
	Data []Ativo `json:"data"`
}

// handleListaAtivos atende GET /api/v1/ativos
func handleListaAtivos(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		ativos, err := listAtivos(ctx, pool)
		if err != nil {
			log.Printf("listar ativos: %v", err) // detalhe so no log
			writeJSON(w, http.StatusInternalServerError, "{\"error\":\"erro interno\"}")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(listaAtivos{Data: ativos}); err != nil {
			log.Printf("erro ao escrever resposta: %v", err)
		}
	}
}

// listAtivos busca ativos no banco
func listAtivos(ctx context.Context, pool *pgxpool.Pool) ([]Ativo, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, identifier, type, criticality, created_at
		FROM assets
		ORDER BY id
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("consultar ativos: %w", err)
	}

	ativos, err := pgx.CollectRows(rows, pgx.RowToStructByName[Ativo])
	if err != nil {
		return nil, fmt.Errorf("ler ativos: %w", err)
	}
	if ativos == nil {
		ativos = []Ativo{} // garante [] no json
	}

	return ativos, nil
}
