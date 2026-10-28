package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

type ativoResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Identifier  string    `json:"identifier"`
	Type        string    `json:"type"`
	Criticality string    `json:"criticality"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"version"`
}

type listaAtivosResponse struct {
	Data []ativoResponse `json:"data"`
}

func toAtivoResponse(a domain.Asset) ativoResponse {
	return ativoResponse{
		ID:          a.ID,
		Name:        a.Name,
		Identifier:  a.Identifier,
		Type:        string(a.Type),
		Criticality: string(a.Criticality),
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Version:     a.Version,
	}
}

func handleListaAtivos(svc *service.AssetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		assets, err := svc.List(ctx)
		if err != nil {
			log.Printf("listar ativos: %v", err)
			writeJSON(w, http.StatusInternalServerError, `{"error":"erro interno"}`)
			return
		}

		resp := listaAtivosResponse{Data: make([]ativoResponse, 0, len(assets))}
		for _, a := range assets {
			resp.Data = append(resp.Data, toAtivoResponse(a))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}
