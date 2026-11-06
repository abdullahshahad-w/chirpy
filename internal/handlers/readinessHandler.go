package handlers

import (
	"log"
	"net/http"
)

func ReadinessHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

	body := []byte("OK")

	_, err := w.Write(body)
	if err != nil {
		log.Println(err)
	}
	
}