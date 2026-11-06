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
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)

	body := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileServerHits.Load())

	_, err := w.Write([]byte(body))
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
	serveMux.HandleFunc("GET /api/healthz", handlers.ReadinessHandler)
	serveMux.HandleFunc("GET /admin/metrics", apiConfig.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", apiConfig.resetHandler)
	serveMux.HandleFunc("POST /api/validate_chirp", handlers.ValidateChirp)

	server := http.Server{
		Addr: ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}