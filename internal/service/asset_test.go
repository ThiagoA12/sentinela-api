package service_test

import (
	"context"
	"errors"
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
}

func (f *fakeAssetStore) List(ctx context.Context) ([]domain.Asset, error) {
	return f.assets, f.err
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

func ptr[T any](v T) *T {
	return &v
}

func TestAssetServiceList(t *testing.T) {
	want := []domain.Asset{
		{ID: uuid.New(), Name: "srv-web-01", Type: domain.AssetTypeServidor},
		{ID: uuid.New(), Name: "portal", Type: domain.AssetTypeAplicacao},
	}
	svc := service.NewAssetService(&fakeAssetStore{assets: want})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("recebeu %d ativos, esperado %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID {
			t.Errorf("ativo %d: ID %v, esperado %v", i, got[i].ID, want[i].ID)
		}
	}
}

func TestAssetServiceListPropagaErro(t *testing.T) {
	errBanco := errors.New("banco fora do ar")
	svc := service.NewAssetService(&fakeAssetStore{err: errBanco})

	_, err := svc.List(context.Background())
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
