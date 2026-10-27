// Package config lê e valida a configuração da API a partir de variáveis de ambiente.
package config

/*
	Importação de biblioteca:
	"errors" - biblioteca padrão pra erros.
	"os" - acesso ao sistema operacional, aqui pra ler variáveis de ambiente.
*/
import (
	"errors"
	"os"
)

// Struct Config reúne tudo que a API precisa pra iniciar.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

// Função para ler variáveis do ambiente, indicando erro.
func getEnv(key, fallback string) string {
	// 'v' recebe o valor de key; 'ok' recebe true/false, confirmando se passou o valor.
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v // Então retorna o valor da key
	}
	return fallback // Variável não existe ou está vazia. Retorna valor padrão.
}

// Load lê o ambiente e devolve erro se faltar algo obrigatório.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getEnv("HTTP_ADDR", "127.0.0.1:8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	// Trata o erro caso a string venha vazia, sem a URL do Banco de Dados.
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL não definida") // Config vazia e o um aviso de erro.
	}

	return cfg, nil // Função retorna Config e o erro como "nil"
}
