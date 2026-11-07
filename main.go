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
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type UserPassword struct {
	Password string `json:"password"`
	Email    string `json:"email"`
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
	type parameters struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Couldn't decode parameters")
		log.Printf("Error decoding parameters: %s", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	cleaned_text := censortextfromjsonbody(params.Body, []string{"kerfuffle", "sharbert", "fornax"}, "****")

	chirp_params := database.CreateChirpParams{
		Body:   cleaned_text,
		UserID: params.UserId,
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

func (cfg *apiConfig) retrieveallchirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbqueries.RetrieveAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "Error retrieving all chirps")
		log.Printf("Error in retrieveallchirpsHandler: %s", err)
		return
	}

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

	respondWithJSON(w, 200, chirp_list)
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
	var u User
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
	}
	respondWithJSON(w, 201, response_user)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	var u User
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

	boolean, err := auth.CheckPasswordHash(u.Password, db_user.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	response_user := User{
		ID:        db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email:     db_user.Email,
	}
	if boolean {
		respondWithJSON(w, 200, response_user)
	} else {
		respondWithError(w, 401, "Incorrect email or password")
	}
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
		dbqueries: dbQueries,
		platform:  os.Getenv("PLATFORM"),
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

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux,
	}

	chripy_server.ListenAndServe()
}
