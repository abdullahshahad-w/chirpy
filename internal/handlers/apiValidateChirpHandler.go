package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"github.com/abdullahshahad-w/chirpy/internal/helpers"
)

func ValidateChirp(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	param := parameters{}
	err := decoder.Decode(&param)
	if err != nil {
		log.Fatal(err)
	}

	if len(param.Body) > 140 {
		helpers.RespondWithError(w, 400, "Chirp is too long")
		return
	}

	profane := map[string]bool{
		"kerfuffle": true,
		"sharbert": true,
		"fornax": true,
	}

	words := strings.Split(param.Body, " ")

	for i := range words {
		if _, ok := profane[strings.ToLower(words[i])]; ok {
			words[i] = "****"
		}
	}

	param.Body = strings.Join(words, " ")

	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
	}

	helpers.RespondWithJSON(w, 200, responseBody{CleanedBody: param.Body})
}