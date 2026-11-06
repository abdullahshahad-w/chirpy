package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/abdullahshahad-w/chirpy/internal/handlers"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

	body := fmt.Appendf(nil, "Hits: %d\n", cfg.fileServerHits.Load())
	
	_, err := w.Write(body)
	if err != nil {
		log.Fatal(err)
	}
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	cfg.fileServerHits.Store(0)
}

func main() {
	serveMux := http.NewServeMux()

	apiConfig := &apiConfig{}

	serveMux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app",http.FileServer(http.Dir("")))))
	serveMux.HandleFunc("GET /healthz", handlers.ReadinessHandler)
	serveMux.HandleFunc("GET /metrics", apiConfig.metricsHandler)
	serveMux.HandleFunc("POST /reset", apiConfig.resetHandler)

	server := http.Server{
		Addr: ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}