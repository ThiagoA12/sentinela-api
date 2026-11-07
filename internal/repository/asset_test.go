package repository

import "testing"

func TestEscapeLike(t *testing.T) {
	if escapeLike(nil) != nil {
		t.Fatal("nil deveria continuar nil")
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"texto comum", "srv-web", "srv-web"},
		{"porcentagem", "100%", `100\%`},
		{"sublinhado", "db_prod", `db\_prod`},
		{"barra invertida", `C:\dados`, `C:\\dados`},
		{"tudo junto", `a\%_`, `a\\\%\_`},
		{"vazio", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			got := escapeLike(&in)
			if got == nil {
				t.Fatalf("escapeLike(%q) = nil, quero %q", tt.in, tt.want)
			}
			if *got != tt.want {
				t.Fatalf("escapeLike(%q) = %q, quero %q", tt.in, *got, tt.want)
			}
		})
	}
}
