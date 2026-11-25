package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

type assetService interface {
	List(ctx context.Context, in service.ListAssetsInput) (service.AssetPage, error)
	Create(ctx context.Context, actor domain.User, in service.CreateAssetInput) (domain.Asset, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Asset, error)
	Update(ctx context.Context, actor domain.User, id uuid.UUID, in service.UpdateAssetInput) (domain.Asset, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type criarAtivoRequest struct {
	Name        string  `json:"name"`
	Identifier  string  `json:"identifier"`
	Type        string  `json:"type"`
	Criticality string  `json:"criticality"`
	Description *string `json:"description"`
	OwnerID     *string `json:"owner_id"`
}

type atualizarAtivoRequest struct {
	Name        *string `json:"name"`
	Identifier  *string `json:"identifier"`
	Type        *string `json:"type"`
	Criticality *string `json:"criticality"`
	Description *string `json:"description"`
	OwnerID     *string `json:"owner_id"`
	Version     *int32  `json:"version"`
}

type ativoResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Identifier  string    `json:"identifier"`
	Type        string    `json:"type"`
	Criticality string    `json:"criticality"`
	Description *string   `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"version"`
}

type paginaResponse struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type listaAtivosResponse struct {
	Data []ativoResponse `json:"data"`
	Page paginaResponse  `json:"page"`
}

func toAtivoResponse(a domain.Asset) ativoResponse {
	return ativoResponse{
		ID:          a.ID,
		Name:        a.Name,
		Identifier:  a.Identifier,
		Type:        string(a.Type),
		Criticality: string(a.Criticality),
		Description: a.Description,
		OwnerID:     a.OwnerID,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   a.CreatedAt.UTC(),
		UpdatedAt:   a.UpdatedAt.UTC(),
		Version:     a.Version,
	}
}

func listaAtivosInput(r *http.Request) (service.ListAssetsInput, error) {
	q, err := parseQuery(r, "page", "page_size", "sort", "type", "criticality", "owner_id", "q")
	if err != nil {
		return service.ListAssetsInput{}, err
	}

	page, err := queryInt(q, "page")
	if err != nil {
		return service.ListAssetsInput{}, err
	}
	pageSize, err := queryInt(q, "page_size")
	if err != nil {
		return service.ListAssetsInput{}, err
	}

	ownerID, err := queryUUID(q, "owner_id")
	if err != nil {
		return service.ListAssetsInput{}, err
	}

	return service.ListAssetsInput{
		Type:        queryString(q, "type"),
		Criticality: queryString(q, "criticality"),
		OwnerID:     ownerID,
		Search:      queryString(q, "q"),
		Sort:        queryString(q, "sort"),
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func handleListaAtivos(svc assetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, err := listaAtivosInput(r)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		p, err := svc.List(ctx, in)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		resp := listaAtivosResponse{
			Data: make([]ativoResponse, 0, len(p.Assets)),
			Page: paginaResponse{Page: p.Page, PageSize: p.PageSize, Total: p.Total},
		}
		for _, a := range p.Assets {
			resp.Data = append(resp.Data, toAtivoResponse(a))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}

func handleCriaAtivo(svc assetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := userFrom(r.Context())
		if !ok {
			writeProblem(w, r, errors.New("usuario ausente no context: rota sem withAuth"))
			return
		}

		var req criarAtivoRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		a, err := svc.Create(ctx, u, service.CreateAssetInput{
			Name:        req.Name,
			Identifier:  req.Identifier,
			Type:        req.Type,
			Criticality: req.Criticality,
			Description: req.Description,
			OwnerID:     req.OwnerID,
		})
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "/api/v1/ativos/"+a.ID.String())
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(toAtivoResponse(a)); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}

func handleBuscaAtivo(svc assetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		a, err := svc.Get(ctx, id)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(toAtivoResponse(a)); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}

func handleAtualizaAtivo(svc assetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := userFrom(r.Context())
		if !ok {
			writeProblem(w, r, errors.New("usuario ausente no context: rota sem withAuth"))
			return
		}

		id, err := pathID(r)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		var req atualizarAtivoRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		a, err := svc.Update(ctx, u, id, service.UpdateAssetInput{
			Name:        req.Name,
			Identifier:  req.Identifier,
			Type:        req.Type,
			Criticality: req.Criticality,
			Description: req.Description,
			OwnerID:     req.OwnerID,
			Version:     req.Version,
		})
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(toAtivoResponse(a)); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}

func handleExcluiAtivo(svc assetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := svc.Delete(ctx, id); err != nil {
			writeProblem(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
