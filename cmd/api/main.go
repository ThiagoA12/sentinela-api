// package main é o ponto de entrada da API
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ThiagoA12/sentinela-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run monta e executa a API. Devolve o erro em vez de encerrar o programa
// para que todos os defer rodem antes da saída.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	//ctx é cancelado quando SIGINT (Ctrl + c) ou SIGTERM (docker compose down, docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	//Cria pool de conexoes, ainda nao conecta so valida a URL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("configurar pool do banco: %w", err)
	}
	defer pool.Close()

	// testando conexao, com prazo de 5s. sem banco a api nao sobe.
	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("conectar ao banco: %w", err)
	}
	log.Println("conectado ao banco")

	mux := http.NewServeMux()

	//aqui esta registrando a rota GET /healthz
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /readyz", handleReadyz(pool))

	//criacao do servidor com as configuracoes: endereco, roteador e timeouts
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,              // quem trata as requisicoes
		ReadHeaderTimeout: 5 * time.Second,  // tempo para o cliente mandar cabeçalhos da requisição
		ReadTimeout:       10 * time.Second, // tempo para ler a requisição inteira (cabeçalhos + corpo)
		WriteTimeout:      15 * time.Second, // tempo para a API escrever a resposta
		IdleTimeout:       60 * time.Second, // tempo para conexão ficar parada esperando próxima requisição
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("API ouvindo em http://%s", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err //ex.: porta ocupada
	case <-ctx.Done():
	}
	stop()

	log.Println("sinal recebido, desligando...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Println("servidor encerrado...")
	return nil
}

// handleHealthz responde se o processo esta ativo. nao consulta banco.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, "{\"status\":\"ok\"}")
}

// handleReadyz responde se a api consegue etender. testa o banco a cada chamada
func handleReadyz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			log.Printf("readyz: banco indisponivel: %v", err) //detalhe so no log
			writeJSON(w, http.StatusServiceUnavailable, "{\"status\":\"unavailable\"}")
			return
		}
		writeJSON(w, http.StatusOK, "{\"status\":\"ok\"}")
	}
}

// writeJSON escreve um corpo JSON pronto com o codigo de status informado
func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
