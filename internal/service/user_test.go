package service_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
	"github.com/ThiagoA12/sentinela-api/internal/auth"
	"github.com/ThiagoA12/sentinela-api/internal/domain"
	"github.com/ThiagoA12/sentinela-api/internal/service"
)

const testAdminPassword = "senha-longa-de-teste"

type fakeUserStore struct {
	count     int64
	countErr  error
	createErr error
	created   *domain.User
}

func (f *fakeUserStore) Count(ctx context.Context) (int64, error) {
	return f.count, f.countErr
}

func (f *fakeUserStore) Create(ctx context.Context, u domain.User) (domain.User, error) {
	if f.createErr != nil {
		return domain.User{}, f.createErr
	}
	f.created = &u
	u.ID = uuid.New()
	u.Active = true
	u.Version = 1
	return u, nil
}

func TestEnsureInitialAdminCreates(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		wantEmail string
	}{
		{"comum", "admin@exemplo.com", testAdminPassword, "admin@exemplo.com"},
		{"espacos em volta do e-mail", "  admin@exemplo.com\n", testAdminPassword, "admin@exemplo.com"},
		{"e-mail com 254 caracteres", strings.Repeat("a", 242) + "@exemplo.com", testAdminPassword, strings.Repeat("a", 242) + "@exemplo.com"},
		{"senha com 15 caracteres", "admin@exemplo.com", strings.Repeat("s", 15), "admin@exemplo.com"},
		{"senha com 128 caracteres", "admin@exemplo.com", strings.Repeat("s", 128), "admin@exemplo.com"},
		{"senha com espacos nas pontas", "admin@exemplo.com", "  senha com espacos  ", "admin@exemplo.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}
			svc := service.NewUserService(store)

			created, err := svc.EnsureInitialAdmin(t.Context(), tt.email, tt.password)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if !created {
				t.Fatal("deveria informar que criou o administrador")
			}
			if store.created == nil {
				t.Fatal("Create nao foi chamado")
			}

			u := *store.created
			if u.Name != "Administrador" || u.Email != tt.wantEmail || u.Role != domain.RoleAdmin {
				t.Errorf("usuario = nome %q, e-mail %q, papel %q; quero Administrador, %q, admin", u.Name, u.Email, u.Role, tt.wantEmail)
			}
			ok, err := auth.VerifyPassword(tt.password, u.PasswordHash)
			if err != nil || !ok {
				t.Errorf("o hash gravado nao confere com a senha: ok=%v, err=%v", ok, err)
			}
		})
	}
}

func TestEnsureInitialAdminUsersExist(t *testing.T) {
	store := &fakeUserStore{count: 1}
	svc := service.NewUserService(store)

	created, err := svc.EnsureInitialAdmin(t.Context(), "", "")
	if err != nil {
		t.Fatalf("com usuarios no banco as variaveis devem ser ignoradas, veio %v", err)
	}
	if created {
		t.Error("nao deveria informar que criou")
	}
	if store.created != nil {
		t.Error("Create nao deveria ser chamado")
	}
}

func TestEnsureInitialAdminInvalid(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantVar  string
	}{
		{"e-mail vazio", "", testAdminPassword, "ADMIN_EMAIL"},
		{"e-mail so com espacos", "   ", testAdminPassword, "ADMIN_EMAIL"},
		{"sem arroba", "admin.exemplo.com", testAdminPassword, "ADMIN_EMAIL"},
		{"nada antes do arroba", "@exemplo.com", testAdminPassword, "ADMIN_EMAIL"},
		{"nada depois do arroba", "admin@", testAdminPassword, "ADMIN_EMAIL"},
		{"dois arrobas", "admin@exemplo@com", testAdminPassword, "ADMIN_EMAIL"},
		{"e-mail com 255 caracteres", strings.Repeat("a", 243) + "@exemplo.com", testAdminPassword, "ADMIN_EMAIL"},
		{"senha vazia", "admin@exemplo.com", "", "ADMIN_PASSWORD"},
		{"senha com 14 caracteres", "admin@exemplo.com", "senha-curta-14", "ADMIN_PASSWORD"},
		{"senha com 129 caracteres", "admin@exemplo.com", strings.Repeat("s", 129), "ADMIN_PASSWORD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}
			svc := service.NewUserService(store)

			created, err := svc.EnsureInitialAdmin(t.Context(), tt.email, tt.password)
			if err == nil {
				t.Fatal("deveria recusar")
			}
			if !strings.Contains(err.Error(), tt.wantVar) {
				t.Errorf("a mensagem deveria citar %s, veio %q", tt.wantVar, err)
			}
			if tt.password != "" && strings.Contains(err.Error(), tt.password) {
				t.Errorf("a mensagem nao pode conter a senha: %q", err)
			}
			if created || store.created != nil {
				t.Error("nada deveria ser criado")
			}
		})
	}
}

func TestEnsureInitialAdminStoreErrors(t *testing.T) {
	errBanco := errors.New("banco fora do ar")

	tests := []struct {
		name  string
		store *fakeUserStore
	}{
		{"falha ao contar", &fakeUserStore{countErr: errBanco}},
		{"falha ao criar", &fakeUserStore{createErr: errBanco}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewUserService(tt.store)

			created, err := svc.EnsureInitialAdmin(t.Context(), "admin@exemplo.com", testAdminPassword)
			if !errors.Is(err, errBanco) {
				t.Fatalf("quero o erro do banco, veio %v", err)
			}
			if created {
				t.Error("nao deveria informar que criou")
			}
		})
	}
}

func validUserInput() service.CreateUserInput {
	return service.CreateUserInput{
		Name:     "Ana",
		Email:    "ana@exemplo.com",
		Password: testAdminPassword,
		Role:     "analista",
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(in *service.CreateUserInput)
		wantName  string
		wantEmail string
		wantRole  domain.Role
	}{
		{"analista", func(in *service.CreateUserInput) {}, "Ana", "ana@exemplo.com", domain.RoleAnalista},
		{"leitor", func(in *service.CreateUserInput) { in.Role = "leitor" }, "Ana", "ana@exemplo.com", domain.RoleLeitor},
		{"admin", func(in *service.CreateUserInput) { in.Role = "admin" }, "Ana", "ana@exemplo.com", domain.RoleAdmin},
		{"espacos em volta do nome", func(in *service.CreateUserInput) { in.Name = "  Ana\t" }, "Ana", "ana@exemplo.com", domain.RoleAnalista},
		{"espacos em volta do e-mail", func(in *service.CreateUserInput) { in.Email = " ana@exemplo.com\n" }, "Ana", "ana@exemplo.com", domain.RoleAnalista},
		{"nome com 100 caracteres acentuados", func(in *service.CreateUserInput) { in.Name = strings.Repeat("ç", 100) }, strings.Repeat("ç", 100), "ana@exemplo.com", domain.RoleAnalista},
		{"senha com 15 caracteres", func(in *service.CreateUserInput) { in.Password = strings.Repeat("s", 15) }, "Ana", "ana@exemplo.com", domain.RoleAnalista},
		{"senha com 128 caracteres", func(in *service.CreateUserInput) { in.Password = strings.Repeat("s", 128) }, "Ana", "ana@exemplo.com", domain.RoleAnalista},
		{"senha com espacos nas pontas", func(in *service.CreateUserInput) { in.Password = "  senha com espacos  " }, "Ana", "ana@exemplo.com", domain.RoleAnalista},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}
			svc := service.NewUserService(store)

			in := validUserInput()
			tt.modify(&in)

			got, err := svc.Create(t.Context(), in)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if store.created == nil {
				t.Fatal("Create nao foi chamado")
			}

			u := *store.created
			if u.Name != tt.wantName || u.Email != tt.wantEmail || u.Role != tt.wantRole {
				t.Errorf("usuario = nome %q, e-mail %q, papel %q; quero %q, %q, %q", u.Name, u.Email, u.Role, tt.wantName, tt.wantEmail, tt.wantRole)
			}
			ok, err := auth.VerifyPassword(in.Password, u.PasswordHash)
			if err != nil || !ok {
				t.Errorf("o hash gravado nao confere com a senha: ok=%v, err=%v", ok, err)
			}
			if got.ID == uuid.Nil {
				t.Error("deveria devolver o usuario criado pelo store")
			}
		})
	}
}

func TestCreateUserValidacao(t *testing.T) {
	const (
		obrigatorio  = "obrigatorio"
		formatoEmail = "use nome@dominio, max 254 caracteres"
		tamanhoSenha = "deve ter entre 15 e 128 caracteres"
		papeis       = "deve ser: admin, analista ou leitor"
	)

	tests := []struct {
		name   string
		modify func(in *service.CreateUserInput)
		want   []apperror.FieldError
	}{
		{"nome vazio", func(in *service.CreateUserInput) { in.Name = "" }, []apperror.FieldError{{Field: "name", Reason: obrigatorio}}},
		{"nome so com espacos", func(in *service.CreateUserInput) { in.Name = "   " }, []apperror.FieldError{{Field: "name", Reason: obrigatorio}}},
		{"nome com 101 caracteres", func(in *service.CreateUserInput) { in.Name = strings.Repeat("a", 101) }, []apperror.FieldError{{Field: "name", Reason: "maximo de 100 caracteres"}}},
		{"e-mail vazio", func(in *service.CreateUserInput) { in.Email = "" }, []apperror.FieldError{{Field: "email", Reason: obrigatorio}}},
		{"e-mail so com espacos", func(in *service.CreateUserInput) { in.Email = "   " }, []apperror.FieldError{{Field: "email", Reason: obrigatorio}}},
		{"e-mail sem arroba", func(in *service.CreateUserInput) { in.Email = "ana.exemplo.com" }, []apperror.FieldError{{Field: "email", Reason: formatoEmail}}},
		{"e-mail com dois arrobas", func(in *service.CreateUserInput) { in.Email = "ana@exemplo@com" }, []apperror.FieldError{{Field: "email", Reason: formatoEmail}}},
		{"e-mail com 255 caracteres", func(in *service.CreateUserInput) { in.Email = strings.Repeat("a", 243) + "@exemplo.com" }, []apperror.FieldError{{Field: "email", Reason: formatoEmail}}},
		{"senha vazia", func(in *service.CreateUserInput) { in.Password = "" }, []apperror.FieldError{{Field: "password", Reason: tamanhoSenha}}},
		{"senha com 14 caracteres", func(in *service.CreateUserInput) { in.Password = "senha-curta-14" }, []apperror.FieldError{{Field: "password", Reason: tamanhoSenha}}},
		{"senha com 129 caracteres", func(in *service.CreateUserInput) { in.Password = strings.Repeat("s", 129) }, []apperror.FieldError{{Field: "password", Reason: tamanhoSenha}}},
		{"papel vazio", func(in *service.CreateUserInput) { in.Role = "" }, []apperror.FieldError{{Field: "role", Reason: papeis}}},
		{"papel desconhecido", func(in *service.CreateUserInput) { in.Role = "dono" }, []apperror.FieldError{{Field: "role", Reason: papeis}}},
		{"papel com maiuscula", func(in *service.CreateUserInput) { in.Role = "Admin" }, []apperror.FieldError{{Field: "role", Reason: papeis}}},
		{"tudo invalido", func(in *service.CreateUserInput) {
			in.Name = ""
			in.Email = "ana"
			in.Password = "curta"
			in.Role = "dono"
		}, []apperror.FieldError{
			{Field: "name", Reason: obrigatorio},
			{Field: "email", Reason: formatoEmail},
			{Field: "password", Reason: tamanhoSenha},
			{Field: "role", Reason: papeis},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}
			svc := service.NewUserService(store)

			in := validUserInput()
			tt.modify(&in)

			_, err := svc.Create(t.Context(), in)

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

func TestCreateUserPropagaErro(t *testing.T) {
	errConflito := apperror.Conflict("ja existe um usuario com esse email", errors.New("23505"))
	svc := service.NewUserService(&fakeUserStore{createErr: errConflito})

	_, err := svc.Create(t.Context(), validUserInput())
	if !errors.Is(err, errConflito) {
		t.Fatalf("erro = %v, esperado %v", err, errConflito)
	}
}
