package main

import (
	"log"
	"net/http"
	"github.com/abdullahshahad-w/chirpy/internal/handlers"
)

func main() {
	serveMux := http.NewServeMux()

	serveMux.Handle("/app/", http.StripPrefix("/app",http.FileServer(http.Dir(""))))
	serveMux.HandleFunc("/healthz", handlers.ReadinessHandler)
	
	server := http.Server{
		Addr: ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}