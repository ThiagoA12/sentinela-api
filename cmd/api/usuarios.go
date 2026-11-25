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

type userService interface {
	Create(ctx context.Context, in service.CreateUserInput) (domain.User, error)
}

type criarUsuarioRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type usuarioResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

func toUsuarioResponse(u domain.User) usuarioResponse {
	return usuarioResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
		Role:  string(u.Role),
	}
}

func handleCriaUsuario(svc userService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req criarUsuarioRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		u, err := svc.Create(ctx, service.CreateUserInput{
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Role:     req.Role,
		})
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(toUsuarioResponse(u)); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}
