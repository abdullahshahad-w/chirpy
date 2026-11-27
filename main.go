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
	"sort"

	"github.com/abdullahshahad-w/chirpy/internal/auth"
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
	platform string
	secret string
	polkaKey string
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
	if cfg.platform != "dev" {
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
		Email string	`json:"email"`
		Password string	`json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	var param parameter
	if err := decoder.Decode(&param); err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong")
		return	
	}

	userEmail := param.Email
	hashedPassword, err := auth.HashPassword(param.Password)
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong hashing the password")
		return
	}

	createdUser, err := cfg.dbQueries.CreateUser(req.Context(), database.CreateUserParams{Email: userEmail, HashedPassword: hashedPassword})
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong writing the database")
		return
	}

	user := userResponse{
		ID: createdUser.ID,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
		Email: createdUser.Email,
		IsChirpyRed: createdUser.IsChirpyRed,
	}

	helpers.RespondWithJSON(w, 201, user)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	
	type parameters struct {
		Body string			`json:"body"`
		UserId uuid.UUID	`json:"user_id"`
	}

	tokenString, err := auth.GetBearerToken(req.Header) 
	if err != nil {
		helpers.RespondWithError(w, 401, "you need a token")
	}

	userIdFromToken, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		helpers.RespondWithError(w, 401, "Invalid token")
		return
	}



	decoder := json.NewDecoder(req.Body)
	var param parameters
	err = decoder.Decode(&param)
	if err != nil {
		helpers.RespondWithError(w, 400, "Something is wrong with the request")
		return
	}

	body := param.Body
	userId := userIdFromToken

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
	authorId := req.URL.Query().Get("author_id")
	sorting := req.URL.Query().Get("sort")
	var fetchedChirps []database.Chirp
	var err error

	if authorId == "" {
		fetchedChirps, err = cfg.dbQueries.GetChirps(req.Context())
	} else {
		userId, err := uuid.Parse(authorId)
		if err != nil {
			helpers.RespondWithError(w, 400, "author id wrong")
			return
		}
		fetchedChirps, err = cfg.dbQueries.GetChirpsByUserId(req.Context(), userId)
	}

	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong fetching chirps")
		return
	}

	var response []chirpResponse

	if sorting == "desc" {
		sort.Slice(fetchedChirps, func(i int, j int) bool {return fetchedChirps[i].CreatedAt.After(fetchedChirps[j].CreatedAt)})
	} else {
		sort.Slice(fetchedChirps, func(i int, j int) bool {return fetchedChirps[i].CreatedAt.Before(fetchedChirps[j].CreatedAt)})
	}
	
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

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, req *http.Request) {
	type paramater struct {
		Email string	`json:"email"`
		Password string	`json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	var params paramater
	if err := decoder.Decode(&params); err != nil {
		helpers.RespondWithError(w, 400, "Could not decode the params")
		return
	}

	userEmail := params.Email
	userPassword := params.Password


	user, err := cfg.dbQueries.GetUserByEmail(req.Context(), userEmail)
	if err != nil {
		helpers.RespondWithError(w, 401, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(userPassword, user.HashedPassword)
	if err != nil {
		helpers.RespondWithError(w, 500, "Something went wrong checking for password match")
		return
	}

	if !match {
		helpers.RespondWithError(w, 401, "Incorrect email or password")
		return
	}

	expiration := time.Duration(3600) * time.Second

	tokenString, err := auth.MakeJWT(user.ID, cfg.secret, expiration)
	if err != nil {
		helpers.RespondWithError(w, 500, "couldn't generate token string")
		return
	}

	refreshToken, err := cfg.dbQueries.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token: auth.MakeRefreshToken(),
		UserID: user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		helpers.RespondWithError(w, 500, "couldn't generate refresh token")
	}

	response := userResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: userEmail,
		TokenString: tokenString,
		RefreshToken: refreshToken,
		IsChirpyRed: user.IsChirpyRed,
	}

	helpers.RespondWithJSON(w, 200, response)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helpers.RespondWithError(w, 401, err.Error())
		return
	}

	refreshTokenFromDB, err := cfg.dbQueries.GetRefreshToken(req.Context(), refreshToken)
	if err != nil {
		helpers.RespondWithError(w, 401, "refresh token not in db")
		return
	}

	expiredAt := refreshTokenFromDB.ExpiresAt
	revoked := refreshTokenFromDB.RevokedAt

	if !time.Now().Before(expiredAt) || revoked.Valid {
		helpers.RespondWithError(w, 401, "refresh token expired or revoked")
		return
	}

	user, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), refreshTokenFromDB.Token)

	expiration := time.Duration(3600) * time.Second

	tokenString, err := auth.MakeJWT(user.ID, cfg.secret, expiration)
	if err != nil {
		helpers.RespondWithError(w, 500, "couldn't generate token string")
		return
	}

	type parameter struct {
		Token string	`json:"token"`
	}

	response := parameter{
		Token: tokenString,
	}

	helpers.RespondWithJSON(w, 200, response)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helpers.RespondWithError(w, 400, err.Error())
		return
	}

	cfg.dbQueries.SetRevoked(req.Context(), database.SetRevokedParams{
		RevokedAt: sql.NullTime{
			Time: time.Now(),
			Valid: true,
		},
		Token: token,
	})

	helpers.RespondWithJSON(w, 204, struct{}{})
}

func (cfg *apiConfig) updatePasswordHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helpers.RespondWithError(w, 401, err.Error())
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		helpers.RespondWithError(w, 401, "Invalid token")
		return
	}

	type parameter struct {
		Email string	`json:"email"`
		Password string	`json:"password"`
	}

	var params parameter
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&params); err != nil {
		helpers.RespondWithError(w, 400, "Send email & pass")
		return
	}

	newEmail := params.Email
	newPassword := params.Password
	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		helpers.RespondWithError(w, 500, "Couldn't hash password")
		return
	}

	userInfo, err := cfg.dbQueries.SetEmailPassword(req.Context(), database.SetEmailPasswordParams{
		Email: newEmail,
		HashedPassword: hashedPassword,
		ID: userId,
	})
	if err != nil {
		helpers.RespondWithError(w, 500, err.Error())
		return
	}

	type responseParam struct {
		Email string	`json:"email"`
	}

	response := responseParam{
		Email:  userInfo,
	}

	helpers.RespondWithJSON(w, 200, response)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helpers.RespondWithError(w, 401, err.Error())
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		helpers.RespondWithError(w, 403, "Invalid token")
		return
	}

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

	if chirp.UserID != userId {
		helpers.RespondWithError(w, 403, "You're not allowed to delete the chirp")
		return
	}

	err = cfg.dbQueries.DeleteChirpById(req.Context(), chirp.ID)
	if err != nil {
		helpers.RespondWithError(w, 500, "Couldn't delete the chirp")
		return
	}

	helpers.RespondWithJSON(w, 204, struct{}{})
}

func (cfg *apiConfig) updateChirpyRed(w http.ResponseWriter, req *http.Request) {
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil {
		helpers.RespondWithError(w, 401, err.Error())
		return
	}

	if cfg.polkaKey != apiKey {
		helpers.RespondWithError(w, 401, "Unauthorized access")
		return
	}

	type parameters struct {
		Event string	`json:"event"`
		Data struct {
			UserId string	`json:"user_id"`
		}	`json:"data"`
	}

	var params parameters
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&params); if err != nil {
		helpers.RespondWithError(w, 400, err.Error())
		return
	}

	event := params.Event
	if event != "user.upgraded" {
		helpers.RespondWithJSON(w, 204, struct{}{})
		return
	}

	userId, err := uuid.Parse(params.Data.UserId)
	if err != nil {
		helpers.RespondWithError(w, 500, err.Error())
		return
	}

	user, err := cfg.dbQueries.GetUserById(req.Context(), userId)
	if err != nil {
		helpers.RespondWithError(w, 404, "User not in db")
		return
	}

	err = cfg.dbQueries.MakeChirpyRed(req.Context(), user.ID)
	if err != nil {
		helpers.RespondWithError(w, 500, err.Error())
		return
	}

	helpers.RespondWithJSON(w, 204, struct{}{})
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
		platform: os.Getenv("PLATFORM"),
		secret: os.Getenv("SECRET"),
		polkaKey: os.Getenv("POLKA_KEY"),
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
	serveMux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	serveMux.HandleFunc("PUT /api/users", apiCfg.updatePasswordHandler)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp)
	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.updateChirpyRed)

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
	TokenString string	`json:"token"`
	RefreshToken string	`json:"refresh_token"`
	IsChirpyRed bool	`json:"is_chirpy_red"`
}

type chirpResponse struct {
	ID        uuid.UUID	`json:"id"`
	CreatedAt time.Time	`json:"created_at"`
	UpdatedAt time.Time	`json:"updated_at"`
	Body      string	`json:"body"`
	UserId    uuid.UUID	`json:"user_id"`
}
