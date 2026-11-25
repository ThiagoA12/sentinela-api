package main

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

type route struct {
	pattern string
	public  bool
	roles   []domain.Role
	handler http.Handler
}

var (
	allRoles   = []domain.Role{domain.RoleAdmin, domain.RoleAnalista, domain.RoleLeitor}
	writeRoles = []domain.Role{domain.RoleAdmin, domain.RoleAnalista}
	adminRoles = []domain.Role{domain.RoleAdmin}
)

func apiRoutes(pool *pgxpool.Pool, authSvc authService, assetSvc assetService, userSvc userService) []route {
	return []route{
		{pattern: "GET /healthz", public: true, handler: http.HandlerFunc(handleHealthz)},
		{pattern: "GET /readyz", public: true, handler: handleReadyz(pool)},
		{pattern: "POST /api/v1/auth/login", public: true, handler: handleLogin(authSvc)},
		{pattern: "GET /api/v1/me", roles: allRoles, handler: http.HandlerFunc(handleMe)},
		{pattern: "POST /api/v1/usuarios", roles: adminRoles, handler: handleCriaUsuario(userSvc)},
		{pattern: "GET /api/v1/ativos", roles: allRoles, handler: handleListaAtivos(assetSvc)},
		{pattern: "POST /api/v1/ativos", roles: writeRoles, handler: handleCriaAtivo(assetSvc)},
		{pattern: "GET /api/v1/ativos/{id}", roles: allRoles, handler: handleBuscaAtivo(assetSvc)},
		{pattern: "PATCH /api/v1/ativos/{id}", roles: writeRoles, handler: handleAtualizaAtivo(assetSvc)},
		{pattern: "DELETE /api/v1/ativos/{id}", roles: adminRoles, handler: handleExcluiAtivo(assetSvc)},
	}
}

func newRouter(authn authenticator, logger *slog.Logger, routes []route) *http.ServeMux {
	mux := http.NewServeMux()
	for _, rt := range routes {
		h := rt.handler
		if !rt.public {
			h = withAuth(authn, logger, rt.roles, h)
		}
		mux.Handle(rt.pattern, h)
	}
	return mux
}
