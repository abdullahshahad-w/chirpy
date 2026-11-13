package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/abdullahshahad-w/chirpy/internal/database"
	"github.com/abdullahshahad-w/chirpy/internal/handlers"
	"github.com/abdullahshahad-w/chirpy/internal/helpers"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries *database.Queries
	Platform string
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
	if cfg.Platform != "dev" {
		helpers.RespondWithError(w, 403, "You do not have the permission")
		return
	}

	cfg.fileServerHits.Store(0)
	
	err := cfg.dbQueries.ResetUsers(req.Context())
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong for reseting")
		return
	}
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	var param parameter
	if err := decoder.Decode(&param); err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong")
		return	
	}

	userEmail := param.Email

	createdUser, err := cfg.dbQueries.CreateUser(req.Context(), userEmail)
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong writing the database")
		return
	}

	user := userResponse{
		ID: createdUser.ID,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
		Email: createdUser.Email,
	}

	helpers.RespondWithJSON(w, 201, user)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	
	type parameters struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	var param parameters
	err := decoder.Decode(&param)
	if err != nil {
		helpers.RespondWithError(w, 400, "Something is wrong with the request")
		return
	}

	body := param.Body
	userId := param.UserId

	profane := map[string]bool{
		"kerfuffle": true,
		"sharbert": true,
		"fornax": true,
	}

	words := strings.Split(body, " ")

	for i := range words {
		if _, ok := profane[strings.ToLower(words[i])]; ok {
			words[i] = "****"
		}
	}

	body = strings.Join(words, " ")
	
	createdChirp, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body: body,
		UserID: userId,
	})
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong creating chirp")
		return
	}

	chirp := chirpResponse{
		ID: createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body: createdChirp.Body,
		UserId: createdChirp.UserID,
	}

	helpers.RespondWithJSON(w, 201, chirp)
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, req *http.Request) {
	fetchedChirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong fetching the chirps")
		return
	}

	var response []chirpResponse

	for _, chirp := range fetchedChirps {
		response = append(response, chirpResponse{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserId: chirp.UserID,
		})
	}

	helpers.RespondWithJSON(w, 200, response)
}

func (cfg *apiConfig) getChirpById(w http.ResponseWriter, req *http.Request) {
	chirpId, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		helpers.RespondWithError(w, 400, "Invalid chirpID")
		return
	}

	chirp, err := cfg.dbQueries.GetChirpById(req.Context(), chirpId)
	if err != nil {
		helpers.RespondWithError(w, 404, "Chirp not in db")
		return
	}

	respond := chirpResponse{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserId: chirp.UserID,
	}

	helpers.RespondWithJSON(w, 200, respond)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbUrl := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		dbQueries: dbQueries,
		Platform: os.Getenv("PLATFORM"),
	}

	serveMux := http.NewServeMux()

	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app",http.FileServer(http.Dir("")))))
	serveMux.HandleFunc("GET /api/healthz", handlers.ReadinessHandler)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	serveMux.HandleFunc("POST /api/users", apiCfg.createUserHandler)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.createChirp)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getAllChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpById)

	server := http.Server{
		Addr: ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string	`json:"email"`
}

type chirpResponse struct {
	ID        uuid.UUID	`json:"id"`
	CreatedAt time.Time	`json:"created_at"`
	UpdatedAt time.Time	`json:"updated_at"`
	Body      string	`json:"body"`
	UserId    uuid.UUID	`json:"user_id"`
}
