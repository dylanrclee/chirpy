package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dylanrclee/chirpy/internal/auth"
	"github.com/dylanrclee/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbqueries      *database.Queries
	platform       string
	secret         string
	polka_secret   string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Chirpyred bool      `json:"is_chirpy_red"`
}

type UserWithPassword struct {
	User
	Password string `json:"password"`
	Expires  int    `json:"expires_in_seconds"`
}

type UserLogin struct {
	User
	Token         string `json:"token"`
	Refresh_token string `json:"refresh_token"`
}

type Upgrade struct {
	Event string `json:"event"`
	Data  struct {
		User_id string `json:"user_id"`
	} `json:"data"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) htmlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	html_text := fmt.Sprintf(
		`<html>
<body>
<h1>Welcome, Chirpy Admin</h1>
<p>Chirpy has been visited %d times!</p>
</body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(html_text))
}

func (cfg *apiConfig) createchirpHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := Chirp{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Couldn't decode parameters")
		log.Printf("Error decoding parameters: %s", err)
		return
	}

	auth_string, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("No authentication header found: %s", err)
		return
	}

	auth_user, err := auth.ValidateJWT(auth_string, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthroized")
		log.Printf("Token string does not equal secret string %s", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	cleaned_text := censortextfromjsonbody(params.Body, []string{"kerfuffle", "sharbert", "fornax"}, "****")

	chirp_params := database.CreateChirpParams{
		Body:   cleaned_text,
		UserID: auth_user,
	}
	db_chirp, err := cfg.dbqueries.CreateChirp(r.Context(), chirp_params)
	if err != nil {
		respondWithError(w, 500, "Couldn't create chirp")
		log.Printf("Error in createchirpHandler sql function: %s", err)
		return
	}

	response_chirp := Chirp{
		ID:        db_chirp.ID,
		CreatedAt: db_chirp.CreatedAt,
		UpdatedAt: db_chirp.UpdatedAt,
		Body:      db_chirp.Body,
		UserID:    db_chirp.UserID,
	}

	respondWithJSON(w, 201, response_chirp)
}

func (cfg *apiConfig) deletechirpHandler(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Error deleting chirp")
		log.Printf("Error retrieving refresh token: %s\n\n", err)
		return
	}

	auth_user, err := auth.ValidateJWT(refresh_token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unable to validate user")
		log.Printf("Error in authenticating user: %s\n\n", err)
		return
	}

	id_uuid, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 401, "Unable to validate user")
		log.Printf("No chirp id provided in request: %s\n\n", err)
		return
	}

	_, err = cfg.dbqueries.RetrieveOneChirp(r.Context(), id_uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp not found")
			log.Printf("No chirp with provided id found: %s\n\n", err)
			return
		} else {
			respondWithError(w, 500, "Unable to delete chirp")
			log.Printf("Error in sql function RetrieveOneChirp: %s\n\n", err)
			return
		}
	}

	deletechirpparams := database.DeleteChirpParams{
		ID:     id_uuid,
		UserID: auth_user,
	}

	deleted_chirp, err := cfg.dbqueries.DeleteChirp(r.Context(), deletechirpparams)
	if err != nil {
		respondWithError(w, 403, "Unable to validate user")
		log.Printf("Error in finding chirp and user combo: %s\n\n", err)
		return
	}
	if deleted_chirp.ID == uuid.Nil {
		respondWithError(w, 403, "Unable to validate user")
		log.Printf("No chirp deleted with DeleteChirp: %s\n\n", err)
		return
	}

	respondWithJSON(w, 204, "Chirp successfully deleted")
}

func (cfg *apiConfig) retrieveallchirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirplistmaker := func(chirps []database.Chirp) []Chirp {
		chirp_list := make([]Chirp, len(chirps))
		for i, chirp := range chirps {
			chirp_list_item := Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
			chirp_list[i] = chirp_list_item
		}
		return chirp_list
	}

	author_id := r.URL.Query().Get("author_id")
	sort_order := r.URL.Query().Get("sort")

	if author_id == "" {
		chirps, err := cfg.dbqueries.RetrieveAllChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, "Error retrieving all chirps")
			log.Printf("Error in retrieveallchirps sql function: %s\n\n", err)
			return
		}

		chirp_list := chirplistmaker(chirps)

		if sort_order != "" {
			if sort_order == "desc" {
				sort.Slice(chirp_list, func(i, j int) bool { return chirp_list[i].CreatedAt.After(chirp_list[j].CreatedAt) })
			}

			if sort_order != "asc" && sort_order != "desc" {
				respondWithError(w, 400, "Error in sort url text")
				log.Printf("Error in url sort text: %s\n\n", err)
				return
			}
		}

		respondWithJSON(w, 200, chirp_list)

	} else {
		user_uuid, err := uuid.Parse(author_id)
		if err != nil {
			respondWithError(w, 500, "Error retrieving chirps")
			log.Printf("Error parsing author_id to uuid: %s\n\n", err)
			return
		}

		chirps, err := cfg.dbqueries.RetrieveAllChirpsfromUser(r.Context(), user_uuid)
		if err != nil {
			respondWithError(w, 500, "Error retrieving all chirps")
			log.Printf("Error in retrieveallchirpsfromuser sql function %s\n\n", err)
			return
		}

		chirp_list := chirplistmaker(chirps)
		respondWithJSON(w, 200, chirp_list)
	}
}

func (cfg *apiConfig) retrieveonechirpHandler(w http.ResponseWriter, r *http.Request) {

	id_string := r.PathValue("chirpID")
	id_uuid, err := uuid.Parse(id_string)
	if err != nil {
		respondWithError(w, 500, "Error searching for chirp")
		log.Printf("Error parsing id string for chirp: %s", err)
		return
	}

	chirp, err := cfg.dbqueries.RetrieveOneChirp(r.Context(), id_uuid)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "Error searching for chirp")
		log.Printf("Error in RetrieveOneChirp function: %s", err)
		return
	}

	response_chirp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	respondWithJSON(w, 200, response_chirp)

}

func (cfg *apiConfig) validateHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body  string `json:"body"`
		Error error  `json:"error"`
		Valid bool   `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Internal error in validation process")
		log.Printf("Error decoding parameters: %s", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	cleaned_text := censortextfromjsonbody(params.Body, []string{"kerfuffle", "sharbert", "fornax"}, "****")

	type payload struct {
		Cleaned_body string `json:"cleaned_body"`
	}
	payload_val := payload{
		Cleaned_body: cleaned_text,
	}
	respondWithJSON(w, 200, payload_val)

}

func (cfg *apiConfig) resetnumHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}

	cfg.fileserverHits.Swap(0)

	err := cfg.dbqueries.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error in DeleteAllUsers sql function: %s\n", err)
		return
	}
	fmt.Printf("Successfully deleted all users\n")
}

func (cfg *apiConfig) createuserHandler(w http.ResponseWriter, r *http.Request) {
	var u UserWithPassword
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		respondWithError(w, 500, "User not created")
		log.Printf("Error in decoding user from request body: %s", err)
		return
	}

	hashed_password, err := auth.HashPassword(u.Password)
	if err != nil {
		respondWithError(w, 500, "User not created")
		log.Printf("Error in hashing password: %s", err)
		return
	}

	upass := database.CreateUserParams{
		Email:          u.Email,
		HashedPassword: hashed_password,
	}

	user, err := cfg.dbqueries.CreateUser(r.Context(), upass)
	if err != nil {
		respondWithError(w, 500, "User not created")
		log.Printf("Error in CreateUser sql function: %s", err)
		return
	}

	response_user := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Chirpyred: user.IsChirpyRed.Bool,
	}
	respondWithJSON(w, 201, response_user)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	var u UserWithPassword
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		respondWithError(w, 500, "Unable to login")
		log.Printf("Error in decoding user from request body: %s", err)
		return
	}

	db_user, err := cfg.dbqueries.SearchUserEmail(r.Context(), u.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(db_user.ID, cfg.secret, time.Duration(60*60)*time.Second)
	if err != nil {
		respondWithError(w, 500, "Unable to login")
		log.Printf("Error in making login token: %s", err)
		return
	}

	boolean, err := auth.CheckPasswordHash(u.Password, db_user.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	refresh_token, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, "Unable to login")
		log.Printf("Error in making refresh token: %s", err)
		return
	}

	token_params := database.CreateTokenParams{
		Token:     refresh_token,
		ExpiresAt: time.Now().Add(time.Duration(24*60) * time.Hour),
		UserID:    db_user.ID,
	}

	db_refresh_token, err := cfg.dbqueries.CreateToken(r.Context(), token_params)
	if err != nil {
		respondWithError(w, 500, "Unable to login")
		log.Printf("Error in adding refresh token to database: %s", err)
		return
	}

	response_user := UserLogin{
		User: User{
			ID:        db_user.ID,
			CreatedAt: db_user.CreatedAt,
			UpdatedAt: db_user.UpdatedAt,
			Email:     db_user.Email,
			Chirpyred: db_user.IsChirpyRed.Bool,
		},
		Token:         token,
		Refresh_token: db_refresh_token.Token,
	}
	if boolean {
		respondWithJSON(w, 200, response_user)
	} else {
		respondWithError(w, 401, "Incorrect email or password")
	}
}

func (cfg *apiConfig) updateuserHandler(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("Error retrieving refresh token: %s\n", err)
		return
	}

	var u UserWithPassword
	err = json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		respondWithError(w, 500, "Unable to update email or password")
		log.Printf("Error decoding body from response: %s\n", err)
		return
	}

	if u.Email == "" || u.Password == "" {
		respondWithError(w, 500, "Unable to update email or password")
		log.Printf("Email or password not in response body: %s\n", err)
		return
	}

	auth_user, err := auth.ValidateJWT(refresh_token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("Refresh token does not equal secret string: %s\n", err)
		return
	}

	hash_pass, err := auth.HashPassword(u.Password)
	if err != nil {
		respondWithError(w, 500, "Unable to update email or password")
		log.Printf("Error in hashing password: %s\n", err)
		return
	}

	update_params := database.UpdateUserEmailandPasswordParams{
		ID:             auth_user,
		Email:          u.Email,
		HashedPassword: hash_pass,
	}

	db_user, err := cfg.dbqueries.UpdateUserEmailandPassword(r.Context(), update_params)
	if err != nil {
		respondWithError(w, 500, "Unable to update email or password")
		log.Printf("Error in updating email and password function: %s", err)
		return
	}

	response_user := User{
		ID:        db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email:     db_user.Email,
		Chirpyred: db_user.IsChirpyRed.Bool,
	}
	respondWithJSON(w, 200, response_user)
}

func (cfg *apiConfig) upgradeuserHandler(w http.ResponseWriter, r *http.Request) {
	api_key, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("Error in getting apikey from request: %s\n\n", err)
		return
	}

	if api_key != cfg.polka_secret {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("APIKey does not equal secret: %s\n\n", err)
		return
	}

	var up Upgrade
	err = json.NewDecoder(r.Body).Decode(&up)
	if err != nil {
		respondWithError(w, 500, "Unable to upgrade user")
		log.Printf("Error decoding response body %s\n\n", err)
		return
	}

	if up.Event != "user.upgraded" {
		respondWithJSON(w, 204, "")
		log.Print("Wrong event in respond body\n\n")
		return
	} else {
		parsed_id, err := uuid.Parse(up.Data.User_id)
		if err != nil {
			respondWithError(w, 500, "Unable to upgrade user")
			log.Printf("Error parsing user id from response: %s\n\n", err)
			return
		}

		_, err = cfg.dbqueries.UpgradeUserChirpyRed(r.Context(), parsed_id)
		if err == sql.ErrNoRows {
			respondWithError(w, 404, "User now found")
			log.Printf("Unable to find user id in database: %s\n\n", err)
			return
		}
		if err != nil {
			respondWithError(w, 500, "Unable to update user")
			log.Printf("Error with UpgradeUserChirpyRed function: %s\n\n", err)
			return
		}

		respondWithJSON(w, 204, "")
	}
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 500, "Unable to login")
		log.Printf("Error retrieving refresh token: %s", err)
		return
	}

	db_refresh_token_info, err := cfg.dbqueries.RetrieveOneToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, 401, "Refresh token does not exist")
		log.Printf("Error finding refresh token in database: %s", err)
		return
	}

	if db_refresh_token_info.ExpiresAt.Before(time.Now()) {
		respondWithError(w, 401, "Refresh token expired")
		log.Printf("Error due to refresh token expiration: %s", err)
		return
	}

	if db_refresh_token_info.RevokedAt.Valid {
		respondWithError(w, 401, "Refresh token revoked")
		log.Printf("Error due to refresh token revoked: %s", err)
		return
	}

	token, err := auth.MakeJWT(db_refresh_token_info.UserID, cfg.secret, time.Duration(60*60)*time.Second)
	if err != nil {
		respondWithError(w, 500, "Unable to make login token")
		log.Printf("Error in making login token: %s", err)
		return
	}

	token_response := TokenResponse{
		Token: token,
	}
	respondWithJSON(w, 200, token_response)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 500, "Internal error")
		log.Printf("Error retrieving refresh token: %s", err)
		return
	}

	db_refresh_token_info, err := cfg.dbqueries.RetrieveOneToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, 401, "Refresh token does not exist")
		log.Printf("Error finding refresh token in database: %s", err)
		return
	}

	_, err = cfg.dbqueries.RevokeRefreshToken(r.Context(), db_refresh_token_info.Token)
	if err != nil {
		respondWithError(w, 500, "Internal error")
		log.Printf("Error revoking refresh token %s", err)
		return
	}

	respondWithJSON(w, 204, "")
}

func censortextfromjsonbody(params string, censored_list []string, replacement_text string) string {
	split_text := strings.Split(params, " ")
	cleaned_slice := make([]string, len(split_text))
	for i, item := range split_text {
		for _, censored_item := range censored_list {
			if strings.ToLower(item) == censored_item {
				cleaned_slice[i] = replacement_text
				break
			} else {
				cleaned_slice[i] = item
			}
		}
	}
	return strings.Join(cleaned_slice, " ")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errors struct {
		Error string `json:"error"`
	}

	respBody := errors{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshaling JSON %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Error creating godotenv: %s", err)
		return
	}
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error creating connection to database: %s", err)
		return
	}
	dbQueries := database.New(db)

	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{
		dbqueries:    dbQueries,
		platform:     os.Getenv("PLATFORM"),
		secret:       os.Getenv("SECRET"),
		polka_secret: os.Getenv("POLKA_KEY"),
	}

	dir := http.Dir(".")
	wrappedhandler := http.StripPrefix("/app", http.FileServer(dir))
	ServeMux.Handle("/app/", apiCfg.middlewareMetricsInc(wrappedhandler))

	ServeMux.HandleFunc("GET /api/healthz", Handler)
	ServeMux.HandleFunc("GET /admin/metrics", apiCfg.htmlHandler)
	ServeMux.HandleFunc("POST /admin/reset", apiCfg.resetnumHandler)
	ServeMux.HandleFunc("POST /api/validate_chirp", apiCfg.validateHandler)
	ServeMux.HandleFunc("POST /api/users", apiCfg.createuserHandler)
	ServeMux.HandleFunc("POST /api/chirps", apiCfg.createchirpHandler)
	ServeMux.HandleFunc("GET /api/chirps", apiCfg.retrieveallchirpsHandler)
	ServeMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.retrieveonechirpHandler)
	ServeMux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	ServeMux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	ServeMux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	ServeMux.HandleFunc("PUT /api/users", apiCfg.updateuserHandler)
	ServeMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deletechirpHandler)
	ServeMux.HandleFunc("POST /api/polka/webhooks", apiCfg.upgradeuserHandler)

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux,
	}

	chripy_server.ListenAndServe()
}
