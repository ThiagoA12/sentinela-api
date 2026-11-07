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

type assetService interface {
	List(ctx context.Context, in service.ListAssetsInput) (service.AssetPage, error)
	Create(ctx context.Context, in service.CreateAssetInput) (domain.Asset, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Asset, error)
	Update(ctx context.Context, id uuid.UUID, in service.UpdateAssetInput) (domain.Asset, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type criarAtivoRequest struct {
	Name        string  `json:"name"`
	Identifier  string  `json:"identifier"`
	Type        string  `json:"type"`
	Criticality string  `json:"criticality"`
	Description *string `json:"description"`
}

type atualizarAtivoRequest struct {
	Name        *string `json:"name"`
	Identifier  *string `json:"identifier"`
	Type        *string `json:"type"`
	Criticality *string `json:"criticality"`
	Description *string `json:"description"`
	Version     *int32  `json:"version"`
}

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
		CreatedAt:   a.CreatedAt.UTC(),
		UpdatedAt:   a.UpdatedAt.UTC(),
		Version:     a.Version,
	}
}

func listaAtivosInput(r *http.Request) (service.ListAssetsInput, error) {
	q, err := parseQuery(r, "page", "page_size", "sort", "type", "criticality", "q")
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

	return service.ListAssetsInput{
		Type:        queryString(q, "type"),
		Criticality: queryString(q, "criticality"),
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
		var req criarAtivoRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		a, err := svc.Create(ctx, service.CreateAssetInput{
			Name:        req.Name,
			Identifier:  req.Identifier,
			Type:        req.Type,
			Criticality: req.Criticality,
			Description: req.Description,
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

		a, err := svc.Update(ctx, id, service.UpdateAssetInput{
			Name:        req.Name,
			Identifier:  req.Identifier,
			Type:        req.Type,
			Criticality: req.Criticality,
			Description: req.Description,
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
