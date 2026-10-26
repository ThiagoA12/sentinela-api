// pacote main, indica que isso vira um executavel
package main

/*
	importacao de bibliotecas.
	log - imprimir
	net/http - pra criacao e manipulacao do server
	time - para definir duracoes
*/
import (
	"log"
	"net/http"
	"time"
)

// ponto de entrada do programa
func main() {
	mux := http.NewServeMux() // cria o roteador, aqui decide qual funcao atende cada rota

	//aqui esta registrando a rota GET /healthz; a funcao anonima "func(w http.ResponseWriter, r *http.Request)" roda a cada requisicao
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")                // avisa que o body e JSON
		w.WriteHeader(http.StatusOK)                                      // status 200
		if _, err := w.Write([]byte("{\"status\":\"ok\"}")); err != nil { //envia o body
			log.Printf("erro ao escrever resposta: %v", err)
		}
	})

	//criacao do servido com as configuracoes: endereco, roteador e timeouts
	srv := &http.Server{
		Addr:              "127.0.0.1:8080", //tocar para ":8080" no docke
		Handler:           mux,              // quem trata as requisicoes
		ReadHeaderTimeout: 5 * time.Second,  // protege contra slowloris
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// liga o servidor
	// so retona se der erro, ex porta ocupada
	log.Printf("API ouvindo em http://%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err) // imprime o erro e encerra com codigo 1
	}
}
