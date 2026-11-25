package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewRouter(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	routes := []route{
		{pattern: "GET /publica", public: true, handler: ok},
		{pattern: "GET /protegida", roles: allRoles, handler: ok},
	}

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCalled bool
	}{
		{"rota publica", "/publica", http.StatusNoContent, false},
		{"rota protegida", "/protegida", http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authn := &fakeAuthenticator{err: errTokenDeTeste}
			rec := httptest.NewRecorder()
			newRouter(authn, slog.New(slog.DiscardHandler), routes).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, quero %d", rec.Code, tt.wantStatus)
			}
			if authn.called != tt.wantCalled {
				t.Errorf("Authenticate chamado = %v, quero %v", authn.called, tt.wantCalled)
			}
		})
	}
}

func TestAPIRoutesPublicas(t *testing.T) {
	var got []string
	for _, rt := range apiRoutes(nil, nil, nil) {
		if rt.public {
			got = append(got, rt.pattern)
		}
	}

	want := []string{"GET /healthz", "GET /readyz", "POST /api/v1/auth/login"}
	if !slices.Equal(got, want) {
		t.Errorf("rotas publicas = %q, quero %q", got, want)
	}
}

func TestAPIRoutesExigemToken(t *testing.T) {
	routes := apiRoutes(nil, nil, nil)

	for _, rt := range routes {
		if rt.public {
			continue
		}
		t.Run(rt.pattern, func(t *testing.T) {
			authn := &fakeAuthenticator{err: errTokenDeTeste}
			h := withRecover(newRouter(authn, slog.New(slog.DiscardHandler), routes))

			method, path, _ := strings.Cut(rt.pattern, " ")
			path = strings.ReplaceAll(path, "{id}", uuid.NewString())
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, quero 401", rec.Code)
			}
		})
	}
}
