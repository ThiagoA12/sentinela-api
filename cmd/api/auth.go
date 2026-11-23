package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ThiagoA12/sentinela-api/internal/auth"
)

type authService interface {
	Login(ctx context.Context, email, password string) (string, error)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func handleLogin(svc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		token, err := svc.Login(ctx, req.Email, req.Password)
		if err != nil {
			writeProblem(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(loginResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   int(auth.AccessTokenTTL.Seconds()),
		}); err != nil {
			log.Printf("escrever resposta: %v", err)
		}
	}
}
