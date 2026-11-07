package service_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

type fakeAssetStore struct {
	assets    []domain.Asset
	err       error
	created   *domain.Asset
	updated   *domain.Asset
	updateErr error
	listed    *domain.AssetFilter
}

func (f *fakeAssetStore) List(ctx context.Context, filter domain.AssetFilter) ([]domain.Asset, int64, error) {
	f.listed = &filter
	return f.assets, int64(len(f.assets)), f.err
}

func (f *fakeAssetStore) Create(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	f.created = &a
	a.ID = uuid.New()
	a.Version = 1
	return a, nil
}

func (f *fakeAssetStore) Get(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	if f.err != nil {
		return domain.Asset{}, f.err
	}
	for _, a := range f.assets {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.Asset{}, apperror.NotFound("ativo nao encontrado")
}

func (f *fakeAssetStore) Update(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	f.updated = &a
	if f.updateErr != nil {
		return domain.Asset{}, f.updateErr
	}
	out := a
	out.Version++
	return out, nil
}

func (f *fakeAssetStore) Delete(ctx context.Context, id uuid.UUID) error {
	return f.err
}

func ptr[T any](v T) *T {
	return &v
}

func TestAssetServiceList(t *testing.T) {
	want := []domain.Asset{
		{ID: uuid.New(), Name: "srv-web-01", Type: domain.AssetTypeServidor},
		{ID: uuid.New(), Name: "portal", Type: domain.AssetTypeAplicacao},
	}
	store := &fakeAssetStore{assets: want}
	svc := service.NewAssetService(store)

	got, err := svc.List(context.Background(), service.ListAssetsInput{})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(got.Assets) != len(want) {
		t.Fatalf("recebeu %d ativos, esperado %d", len(got.Assets), len(want))
	}
	for i := range want {
		if got.Assets[i].ID != want[i].ID {
			t.Errorf("ativo %d: ID %v, esperado %v", i, got.Assets[i].ID, want[i].ID)
		}
	}
	if got.Page != 1 || got.PageSize != 20 || got.Total != 2 {
		t.Errorf("pagina = %d, tamanho = %d, total = %d; esperado 1, 20, 2", got.Page, got.PageSize, got.Total)
	}

	wantFilter := domain.AssetFilter{Sort: "created_at", Page: 1, PageSize: 20}
	if store.listed == nil || *store.listed != wantFilter {
		t.Errorf("filtro recebido pelo store = %+v, esperado %+v", store.listed, wantFilter)
	}
}

func TestAssetServiceListPropagaErro(t *testing.T) {
	errBanco := errors.New("banco fora do ar")
	svc := service.NewAssetService(&fakeAssetStore{err: errBanco})

	_, err := svc.List(context.Background(), service.ListAssetsInput{})
	if !errors.Is(err, errBanco) {
		t.Fatalf("erro = %v, esperado que contivesse %v", err, errBanco)
	}
}

func TestAssetServiceCreate(t *testing.T) {
	store := &fakeAssetStore{}
	svc := service.NewAssetService(store)

	in := service.CreateAssetInput{
		Name:        "  srv-web-01  ",
		Identifier:  "\t10.0.0.10\n",
		Type:        "servidor",
		Criticality: "alta",
		Description: ptr("  servidor web principal  "),
	}

	got, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if store.created == nil {
		t.Fatal("store.Create nao foi chamado")
	}

	c := store.created
	if c.Name != "srv-web-01" {
		t.Errorf("name = %q, esperado %q", c.Name, "srv-web-01")
	}
	if c.Identifier != "10.0.0.10" {
		t.Errorf("identifier = %q, esperado %q", c.Identifier, "10.0.0.10")
	}
	if c.Type != domain.AssetTypeServidor {
		t.Errorf("type = %q, esperado %q", c.Type, domain.AssetTypeServidor)
	}
	if c.Criticality != domain.CriticalityAlta {
		t.Errorf("criticality = %q, esperado %q", c.Criticality, domain.CriticalityAlta)
	}
	if c.Description == nil || *c.Description != "servidor web principal" {
		t.Errorf("description = %v, esperado %q", c.Description, "servidor web principal")
	}
	if got.ID == uuid.Nil {
		t.Error("ID vazio no ativo devolvido")
	}
}

func validInput() service.CreateAssetInput {
	return service.CreateAssetInput{
		Name:        "srv-web-01",
		Identifier:  "10.0.0.10",
		Type:        "servidor",
		Criticality: "alta",
		Description: ptr("servidor web principal"),
	}
}

func TestAssetServiceCreateValidacao(t *testing.T) {
	tests := []struct {
		name   string
		modify func(in *service.CreateAssetInput)
		want   []apperror.FieldError
	}{
		{
			name:   "name com 120 caracteres",
			modify: func(in *service.CreateAssetInput) { in.Name = strings.Repeat("a", 120) },
		},
		{
			name:   "name com 120 caracteres acentuados",
			modify: func(in *service.CreateAssetInput) { in.Name = strings.Repeat("ç", 120) },
		},
		{
			name:   "name vazio",
			modify: func(in *service.CreateAssetInput) { in.Name = "" },
			want:   []apperror.FieldError{{Field: "name", Reason: "obrigatorio"}},
		},
		{
			name:   "name so com espacos",
			modify: func(in *service.CreateAssetInput) { in.Name = "   " },
			want:   []apperror.FieldError{{Field: "name", Reason: "obrigatorio"}},
		},
		{
			name:   "name com 121 caracteres",
			modify: func(in *service.CreateAssetInput) { in.Name = strings.Repeat("a", 121) },
			want:   []apperror.FieldError{{Field: "name", Reason: "maximo de 120 caracteres"}},
		},
		{
			name:   "type desconhecido",
			modify: func(in *service.CreateAssetInput) { in.Type = "impressora" },
			want: []apperror.FieldError{{
				Field:  "type",
				Reason: "deve ser: servidor, aplicacao, rede, estacao ou nuvem",
			}},
		},
		{
			name:   "criticality em maiuscula",
			modify: func(in *service.CreateAssetInput) { in.Criticality = "ALTA" },
			want: []apperror.FieldError{{
				Field:  "criticality",
				Reason: "deve ser: baixa, media, alta ou critica",
			}},
		},
		{
			name:   "description com 2001 caracteres",
			modify: func(in *service.CreateAssetInput) { in.Description = ptr(strings.Repeat("a", 2001)) },
			want:   []apperror.FieldError{{Field: "description", Reason: "maximo de 2000 caracteres"}},
		},
		{
			name: "varios erros de uma vez",
			modify: func(in *service.CreateAssetInput) {
				in.Name = ""
				in.Identifier = ""
				in.Type = ""
				in.Criticality = ""
				in.Description = ptr(strings.Repeat("a", 2001))
			},
			want: []apperror.FieldError{
				{Field: "name", Reason: "obrigatorio"},
				{Field: "identifier", Reason: "obrigatorio"},
				{Field: "type", Reason: "deve ser: servidor, aplicacao, rede, estacao ou nuvem"},
				{Field: "criticality", Reason: "deve ser: baixa, media, alta ou critica"},
				{Field: "description", Reason: "maximo de 2000 caracteres"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeAssetStore{}
			svc := service.NewAssetService(store)

			in := validInput()
			tt.modify(&in)

			_, err := svc.Create(context.Background(), in)

			if tt.want == nil {
				if err != nil {
					t.Fatalf("erro inesperado: %v", err)
				}
				return
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
				t.Fatalf("erro = %v, esperado erro de validacao", err)
			}
			if !slices.Equal(appErr.Fields, tt.want) {
				t.Errorf("fields = %v, esperado %v", appErr.Fields, tt.want)
			}
			if store.created != nil {
				t.Error("store.Create foi chamado com entrada invalida")
			}
		})
	}
}

func TestAssetServiceCreatePropagaErro(t *testing.T) {
	errConflito := apperror.Conflict("ja existe um ativo com este identificador", errors.New("23505"))
	svc := service.NewAssetService(&fakeAssetStore{err: errConflito})

	_, err := svc.Create(context.Background(), validInput())
	if !errors.Is(err, errConflito) {
		t.Fatalf("erro = %v, esperado %v", err, errConflito)
	}
	if apperror.KindOf(err) != apperror.KindConflict {
		t.Errorf("KinkOf = %v, esperado KindConflict", apperror.KindOf(err))
	}
}

func ativoAtual() domain.Asset {
	return domain.Asset{
		ID:          uuid.MustParse("019a3c1e-0000-7000-8000-000000000001"),
		Name:        "Servidor web",
		Identifier:  "srv-web-01",
		Type:        domain.AssetTypeServidor,
		Criticality: domain.CriticalityAlta,
		Description: ptr("fica na DMZ"),
		Version:     3,
	}
}

func TestAssetServiceUpdate(t *testing.T) {
	atual := ativoAtual()
	store := &fakeAssetStore{assets: []domain.Asset{atual}}
	svc := service.NewAssetService(store)

	got, err := svc.Update(context.Background(), atual.ID, service.UpdateAssetInput{
		Criticality: ptr("media"),
		Description: ptr("   "),
		Version:     ptr(int32(3)),
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if store.updated == nil {
		t.Fatalf("store.Update nao foi chamado")
	}

	want := atual
	want.Criticality = domain.CriticalityMedia
	want.Description = nil

	if *store.updated != want {
		t.Errorf("store recebeu %+v, esperado %+v", *store.updated, want)
	}
	if got.Version != 4 {
		t.Errorf("version devolvida = %d, esperado 4", got.Version)
	}
}

func TestAssetServiceUpdateErros(t *testing.T) {
	atual := ativoAtual()
	conflito := apperror.Conflict("ativo alterado por outra requisicao", nil)

	tests := []struct {
		name       string
		id         uuid.UUID
		in         service.UpdateAssetInput
		updateErr  error
		wantKind   apperror.Kind
		wantField  string
		wantUpdate bool
	}{
		{"sem version", atual.ID, service.UpdateAssetInput{Name: ptr("Novo")}, nil, apperror.KindValidation, "version", false},
		{"name so com espacos", atual.ID, service.UpdateAssetInput{Name: ptr("   "), Version: ptr(int32(3))}, nil, apperror.KindValidation, "name", false},
		{"type desconhecido", atual.ID, service.UpdateAssetInput{Type: ptr("impressora"), Version: ptr(int32(3))}, nil, apperror.KindValidation, "type", false},
		{"ativo inexistente", uuid.MustParse("019a3c1e-0000-7000-8000-000000000099"), service.UpdateAssetInput{Version: ptr(int32(3))}, nil, apperror.KindNotFound, "", false},
		{"versao desatualizada", atual.ID, service.UpdateAssetInput{Name: ptr("Novo"), Version: ptr(int32(2))}, conflito, apperror.KindConflict, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeAssetStore{assets: []domain.Asset{atual}, updateErr: tt.updateErr}
			svc := service.NewAssetService(store)

			_, err := svc.Update(context.Background(), tt.id, tt.in)

			if got := apperror.KindOf(err); got != tt.wantKind {
				t.Fatalf("Kind = %v, esperado %v (erro: %v)", got, tt.wantKind, err)
			}
			if tt.wantField != "" {
				var appErr *apperror.Error
				if !errors.As(err, &appErr) {
					t.Fatalf("erro nao e *apperror.Error: %v", err)
				}
				if len(appErr.Fields) != 1 || appErr.Fields[0].Field != tt.wantField {
					t.Errorf("fields = %+v, esperado so %q", appErr.Fields, tt.wantField)
				}
			}
			if chamado := store.updated != nil; chamado != tt.wantUpdate {
				t.Errorf("store.Update chamado = %v, esperado %v", chamado, tt.wantUpdate)
			}
		})
	}
}

func TestAssetServiceListFiltro(t *testing.T) {
	store := &fakeAssetStore{}
	svc := service.NewAssetService(store)

	in := service.ListAssetsInput{
		Type:        ptr("servidor"),
		Criticality: ptr("alta"),
		Search:      ptr("  db_  "),
		Sort:        ptr("-criticality"),
		Page:        ptr(3),
		PageSize:    ptr(50),
	}
	got, err := svc.List(context.Background(), in)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got.Page != 3 || got.PageSize != 50 {
		t.Errorf("pagina = %d, tamanho = %d; esperado 3, 50", got.Page, got.PageSize)
	}

	want := domain.AssetFilter{
		Type:        ptr(domain.AssetTypeServidor),
		Criticality: ptr(domain.CriticalityAlta),
		Search:      ptr("db_"),
		Sort:        "-criticality",
		Page:        3,
		PageSize:    50,
	}
	if store.listed == nil || !reflect.DeepEqual(*store.listed, want) {
		t.Errorf("filtro recebido pelo store = %+v, esperado %+v", store.listed, want)
	}
}

func TestAssetServiceListBuscaEmBranco(t *testing.T) {
	store := &fakeAssetStore{}
	svc := service.NewAssetService(store)

	if _, err := svc.List(context.Background(), service.ListAssetsInput{Search: ptr("   ")}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if store.listed == nil || store.listed.Search != nil {
		t.Errorf("busca em branco deveria virar nil, veio %+v", store.listed)
	}
}

func TestAssetServiceListValidacao(t *testing.T) {
	tests := []struct {
		name       string
		in         service.ListAssetsInput
		wantFields []string
	}{
		{"page 1 (borda)", service.ListAssetsInput{Page: ptr(1)}, nil},
		{"page 10000 (borda)", service.ListAssetsInput{Page: ptr(10000)}, nil},
		{"page_size 1 (borda)", service.ListAssetsInput{PageSize: ptr(1)}, nil},
		{"page_size 100 (borda)", service.ListAssetsInput{PageSize: ptr(100)}, nil},
		{"q com 100 caracteres", service.ListAssetsInput{Search: ptr(strings.Repeat("a", 100))}, nil},
		{"sort crescente", service.ListAssetsInput{Sort: ptr("name")}, nil},
		{"sort decrescente", service.ListAssetsInput{Sort: ptr("-updated_at")}, nil},
		{"page 0", service.ListAssetsInput{Page: ptr(0)}, []string{"page"}},
		{"page negativa", service.ListAssetsInput{Page: ptr(-1)}, []string{"page"}},
		{"page 10001", service.ListAssetsInput{Page: ptr(10001)}, []string{"page"}},
		{"page_size 0", service.ListAssetsInput{PageSize: ptr(0)}, []string{"page_size"}},
		{"page_size 101", service.ListAssetsInput{PageSize: ptr(101)}, []string{"page_size"}},
		{"page_size 1000 (RS-24)", service.ListAssetsInput{PageSize: ptr(1000)}, []string{"page_size"}},
		{"type fora da lista", service.ListAssetsInput{Type: ptr("impressora")}, []string{"type"}},
		{"type vazio", service.ListAssetsInput{Type: ptr("")}, []string{"type"}},
		{"criticality fora da lista", service.ListAssetsInput{Criticality: ptr("urgente")}, []string{"criticality"}},
		{"q com 101 caracteres", service.ListAssetsInput{Search: ptr(strings.Repeat("a", 101))}, []string{"q"}},
		{"sort fora da allowlist", service.ListAssetsInput{Sort: ptr("senha")}, []string{"sort"}},
		{"sort com sql", service.ListAssetsInput{Sort: ptr("name; drop table assets")}, []string{"sort"}},
		{"sort so com hifen", service.ListAssetsInput{Sort: ptr("-")}, []string{"sort"}},
		{"sort com dois hifens", service.ListAssetsInput{Sort: ptr("--name")}, []string{"sort"}},
		{"varios de uma vez", service.ListAssetsInput{
			Page:     ptr(0),
			PageSize: ptr(1000),
			Sort:     ptr("senha"),
		}, []string{"page", "page_size", "sort"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeAssetStore{}
			svc := service.NewAssetService(store)

			_, err := svc.List(context.Background(), tt.in)

			if tt.wantFields == nil {
				if err != nil {
					t.Fatalf("erro inesperado: %v", err)
				}
				if store.listed == nil {
					t.Fatal("store deveria ter sido chamado")
				}
				return
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
				t.Fatalf("erro = %v, esperado erro de validacao", err)
			}
			var got []string
			for _, f := range appErr.Fields {
				got = append(got, f.Field)
			}
			if !slices.Equal(got, tt.wantFields) {
				t.Errorf("campos = %v, esperado %v", got, tt.wantFields)
			}
			if store.listed != nil {
				t.Error("store nao deveria ter sido chamado")
			}
		})
	}
}
