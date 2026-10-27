// package main é o ponto de entrada da API
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	mux := http.NewServeMux()

	//aqui esta registrando a rota GET /healthz
	mux.HandleFunc("GET /healthz", handleHealthz)

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

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("{\"status\":\"ok\"}")); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}
