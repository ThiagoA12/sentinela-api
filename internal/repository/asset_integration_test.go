//go:build integration

package repository

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
)

const postgresImage = "postgres:18"

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(runIntegration(m))
}

func runIntegration(m *testing.M) int {
	ctx := context.Background()

	migrations, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		log.Printf("procurar migration: %v", err)
		return 1
	}
	if len(migrations) == 0 {
		log.Printf("nenhuma migration encontrada")
		return 1
	}

	ctr, err := postgres.Run(ctx, postgresImage,
		postgres.WithOrderedInitScripts(migrations...),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("derrubar container: %v", err)
		}
	}()
	if err != nil {
		log.Printf("subir container: %v", err)
		return 1
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("connection string: %v", err)
		return 1
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Printf("criar pool: %v", err)
		return 1
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Printf("ping no banco de teste: %v", err)
		return 1
	}

	testPool = pool
	return m.Run()
}

func setupAssets(t *testing.T) *AssetRepository {
	t.Helper()
	if _, err := testPool.Exec(t.Context(), "TRUNCATE assets"); err != nil {
		t.Fatalf("limpar assets: %v", err)
	}
	return NewAssetRepository(testPool)
}

func sampleAsset(identifier string) domain.Asset {
	desc := "ativo de teste"
	return domain.Asset{
		Name:        "Servidor " + identifier,
		Identifier:  identifier,
		Type:        domain.AssetTypeServidor,
		Criticality: domain.CriticalityAlta,
		Description: &desc,
	}
}

func mustCreate(t *testing.T, repo *AssetRepository, a domain.Asset) domain.Asset {
	t.Helper()
	created, err := repo.Create(t.Context(), a)
	if err != nil {
		t.Fatalf("criar %q: %v", a.Identifier, err)
	}
	return created
}

func TestAssetRepositoryCreateGet(t *testing.T) {
	repo := setupAssets(t)
	in := sampleAsset("srv-01")

	created := mustCreate(t, repo, in)

	if created.ID.Version() != 7 {
		t.Errorf("id deveria ser UUIDv7, veio versao %d", created.ID.Version())
	}
	if created.CreatedAt.IsZero() || !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Errorf("datas inesperadas: created_at %v, updated_at %v", created.CreatedAt, created.UpdatedAt)
	}

	want := in
	want.ID = created.ID
	want.CreatedAt = created.CreatedAt
	want.UpdatedAt = created.UpdatedAt
	want.Version = 1
	if !reflect.DeepEqual(created, want) {
		t.Errorf("Create = %+v\nquero %+v", created, want)
	}

	got, err := repo.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("Get = %+v\nquero %+v", got, created)
	}
}

func TestAssetRepositoryCreateDuplicate(t *testing.T) {
	repo := setupAssets(t)
	mustCreate(t, repo, sampleAsset("srv-01"))

	_, err := repo.Create(t.Context(), sampleAsset("srv-01"))
	if apperror.KindOf(err) != apperror.KindConflict {
		t.Fatalf("identificador repetido: quero KindConflict, veio %v", err)
	}
}

func TestAssetRepositoryCreateAfterDelete(t *testing.T) {
	repo := setupAssets(t)
	old := mustCreate(t, repo, sampleAsset("srv-01"))
	if err := repo.Delete(t.Context(), old.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}

	again := mustCreate(t, repo, sampleAsset("srv-01"))
	if again.ID == old.ID {
		t.Fatalf("recadastro deveria gerar outro id")
	}
}

func TestAssetRepositoryGetNotFound(t *testing.T) {
	repo := setupAssets(t)
	deleted := mustCreate(t, repo, sampleAsset("srv-01"))
	if err := repo.Delete(t.Context(), deleted.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}

	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{"inexistente", uuid.New()},
		{"excluido", deleted.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Get(t.Context(), tt.id)
			if apperror.KindOf(err) != apperror.KindNotFound {
				t.Fatalf("quero KindNotFound, veio %v", err)
			}
		})
	}
}

func TestAssetRepositoryUpdate(t *testing.T) {
	repo := setupAssets(t)
	created := mustCreate(t, repo, sampleAsset("srv-01"))

	changed := created
	changed.Name = "Servidor renomeado"
	changed.Criticality = domain.CriticalityCritica
	changed.Description = nil

	updated, err := repo.Update(t.Context(), changed)
	if err != nil {
		t.Fatalf("atualizar: %v", err)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at deveria avancar: antes %v, depois %v", created.UpdatedAt, updated.UpdatedAt)
	}

	want := changed
	want.Version = created.Version + 1
	want.UpdatedAt = updated.UpdatedAt
	if !reflect.DeepEqual(updated, want) {
		t.Errorf("Update = %+v\nquero %+v", updated, want)
	}

	got, err := repo.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if !reflect.DeepEqual(got, updated) {
		t.Errorf("Get = %+v\nquero %+v", got, updated)
	}
}

func TestAssetRepositoryUpdateConflict(t *testing.T) {
	repo := setupAssets(t)
	first := mustCreate(t, repo, sampleAsset("srv-01"))
	second := mustCreate(t, repo, sampleAsset("srv-02"))
	deleted := mustCreate(t, repo, sampleAsset("srv-03"))
	if err := repo.Delete(t.Context(), deleted.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}
	if _, err := repo.Update(t.Context(), first); err != nil {
		t.Fatalf("primeira atualizacao: %v", err)
	}

	taken := second
	taken.Identifier = first.Identifier

	missing := sampleAsset("srv-04")
	missing.ID = uuid.New()
	missing.Version = 1

	gone := deleted
	gone.Version = deleted.Version + 1

	tests := []struct {
		name  string
		asset domain.Asset
	}{
		{"versao antiga", first},
		{"identificador de outro ativo", taken},
		{"inexistente", missing},
		{"excluido", gone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Update(t.Context(), tt.asset)
			if apperror.KindOf(err) != apperror.KindConflict {
				t.Fatalf("quero KindConflict, veio %v", err)
			}
		})
	}
}

func TestAssetRepositoryDelete(t *testing.T) {
	repo := setupAssets(t)
	created := mustCreate(t, repo, sampleAsset("srv-01"))

	if err := repo.Delete(t.Context(), created.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}

	var deletedAt *time.Time
	var version int32
	err := testPool.QueryRow(t.Context(),
		"SELECT deleted_at, version FROM assets WHERE id = $1", created.ID).Scan(&deletedAt, &version)
	if err != nil {
		t.Fatalf("a linha deveria continuar na tabela: %v", err)
	}
	if deletedAt == nil {
		t.Errorf("deleted_at deveria estar preenchido")
	}
	if version != created.Version+1 {
		t.Errorf("version = %d, quero %d", version, created.Version+1)
	}

	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{"de novo", created.ID},
		{"inexistente", uuid.New()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(t.Context(), tt.id)
			if apperror.KindOf(err) != apperror.KindNotFound {
				t.Fatalf("quero KindNotFound, veio %v", err)
			}
		})
	}
}

func TestAssetRepositoryConstraints(t *testing.T) {
	repo := setupAssets(t)

	tests := []struct {
		name     string
		change   func(a *domain.Asset)
		wantCode string
	}{
		{"tipo fora da lista", func(a *domain.Asset) { a.Type = "impressora" }, "23514"},
		{"criticidade fora da lista", func(a *domain.Asset) { a.Criticality = "urgente" }, "23514"},
		{"nome com 121 caracteres", func(a *domain.Asset) { a.Name = strings.Repeat("a", 121) }, "22001"},
		{"identificador com 256 caracteres", func(a *domain.Asset) { a.Identifier = strings.Repeat("a", 256) }, "22001"},
		{"descricao com 2001 caracteres", func(a *domain.Asset) {
			d := strings.Repeat("a", 2001)
			a.Description = &d
		}, "22001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := sampleAsset("srv-01")
			tt.change(&a)

			_, err := repo.Create(t.Context(), a)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.wantCode {
				t.Fatalf("quero erro %s do postgres, veio %v", tt.wantCode, err)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestAssetRepositoryList(t *testing.T) {
	repo := setupAssets(t)

	specs := []domain.Asset{
		{Name: "Portal Web", Identifier: "web-01", Type: domain.AssetTypeAplicacao, Criticality: domain.CriticalityCritica},
		{Name: "Banco Principal", Identifier: "db-01", Type: domain.AssetTypeServidor, Criticality: domain.CriticalityAlta},
		{Name: "Firewall Borda", Identifier: "fw-01", Type: domain.AssetTypeRede, Criticality: domain.CriticalityMedia},
		{Name: "Estacao Recepcao", Identifier: "pc-01", Type: domain.AssetTypeEstacao, Criticality: domain.CriticalityBaixa},
		{Name: "Logs 100% nuvem", Identifier: "s3_logs", Type: domain.AssetTypeNuvem, Criticality: domain.CriticalityAlta},
		{Name: "Copia de Dados", Identifier: `bkp\dados`, Type: domain.AssetTypeServidor, Criticality: domain.CriticalityBaixa},
	}
	seeded := make(map[string]domain.Asset, len(specs))
	for _, a := range specs {
		seeded[a.Identifier] = mustCreate(t, repo, a)
	}
	removed := mustCreate(t, repo, domain.Asset{
		Name: "Antigo Removido", Identifier: "old-01", Type: domain.AssetTypeServidor, Criticality: domain.CriticalityAlta,
	})
	if err := repo.Delete(t.Context(), removed.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}

	byID := func(identifiers ...string) []string {
		out := slices.Clone(identifiers)
		slices.SortFunc(out, func(a, b string) int {
			return strings.Compare(seeded[a].ID.String(), seeded[b].ID.String())
		})
		return out
	}

	tests := []struct {
		name      string
		filter    domain.AssetFilter
		want      []string
		wantTotal int64
	}{
		{"created_at", domain.AssetFilter{},
			[]string{"web-01", "db-01", "fw-01", "pc-01", "s3_logs", `bkp\dados`}, 6},
		{"created_at decrescente", domain.AssetFilter{Sort: "-created_at"},
			[]string{`bkp\dados`, "s3_logs", "pc-01", "fw-01", "db-01", "web-01"}, 6},
		{"name", domain.AssetFilter{Sort: "name"},
			[]string{"db-01", `bkp\dados`, "pc-01", "fw-01", "s3_logs", "web-01"}, 6},
		{"name decrescente", domain.AssetFilter{Sort: "-name"},
			[]string{"web-01", "s3_logs", "fw-01", "pc-01", `bkp\dados`, "db-01"}, 6},
		{"identifier", domain.AssetFilter{Sort: "identifier"},
			[]string{`bkp\dados`, "db-01", "fw-01", "pc-01", "s3_logs", "web-01"}, 6},
		{"type com desempate por id", domain.AssetFilter{Sort: "type"},
			slices.Concat([]string{"web-01", "pc-01", "s3_logs", "fw-01"}, byID("db-01", `bkp\dados`)), 6},
		{"criticality por gravidade", domain.AssetFilter{Sort: "criticality"},
			slices.Concat(byID("pc-01", `bkp\dados`), []string{"fw-01"}, byID("db-01", "s3_logs"), []string{"web-01"}), 6},
		{"criticality decrescente", domain.AssetFilter{Sort: "-criticality"},
			slices.Concat([]string{"web-01"}, byID("db-01", "s3_logs"), []string{"fw-01"}, byID("pc-01", `bkp\dados`)), 6},

		{"pagina 1", domain.AssetFilter{Page: 1, PageSize: 2}, []string{"web-01", "db-01"}, 6},
		{"pagina 3", domain.AssetFilter{Page: 3, PageSize: 2}, []string{"s3_logs", `bkp\dados`}, 6},
		{"pagina depois do fim", domain.AssetFilter{Page: 4, PageSize: 2}, nil, 6},
		{"filtro com paginacao", domain.AssetFilter{Type: ptr(domain.AssetTypeServidor), Page: 2, PageSize: 1},
			[]string{`bkp\dados`}, 2},

		{"tipo", domain.AssetFilter{Type: ptr(domain.AssetTypeServidor)}, []string{"db-01", `bkp\dados`}, 2},
		{"criticidade", domain.AssetFilter{Criticality: ptr(domain.CriticalityAlta)}, []string{"db-01", "s3_logs"}, 2},
		{"tipo e criticidade", domain.AssetFilter{
			Type: ptr(domain.AssetTypeServidor), Criticality: ptr(domain.CriticalityBaixa),
		}, []string{`bkp\dados`}, 1},
		{"busca sem diferenciar maiusculas", domain.AssetFilter{Search: ptr("WEB")}, []string{"web-01"}, 1},
		{"busca por porcentagem", domain.AssetFilter{Search: ptr("%")}, []string{"s3_logs"}, 1},
		{"busca por sublinhado", domain.AssetFilter{Search: ptr("_")}, []string{"s3_logs"}, 1},
		{"busca por barra invertida", domain.AssetFilter{Search: ptr(`\`)}, []string{`bkp\dados`}, 1},
		{"busca sem resultado", domain.AssetFilter{Search: ptr("zzz")}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.filter
			if f.Sort == "" {
				f.Sort = "created_at"
			}
			if f.Page == 0 {
				f.Page = 1
			}
			if f.PageSize == 0 {
				f.PageSize = 100
			}

			assets, total, err := repo.List(t.Context(), f)
			if err != nil {
				t.Fatalf("listar: %v", err)
			}
			got := make([]string, 0, len(assets))
			for _, a := range assets {
				got = append(got, a.Identifier)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ordem = %q, quero %q", got, tt.want)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, quero %d", total, tt.wantTotal)
			}
		})
	}
}
