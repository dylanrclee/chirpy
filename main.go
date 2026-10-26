package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
		log.Printf("Error decoding parameters: %s", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else {
		type payload struct {
			Valid bool `json:"valid"`
		}
		payload_val := payload{
			Valid: true,
		}
		respondWithJSON(w, 200, payload_val)
	}
}

func (cfg *apiConfig) resetnumHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Swap(0)
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
	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{}

	dir := http.Dir(".")
	wrappedhandler := http.StripPrefix("/app", http.FileServer(dir))
	ServeMux.Handle("/app/", apiCfg.middlewareMetricsInc(wrappedhandler))

	ServeMux.HandleFunc("GET /api/healthz", Handler)
	ServeMux.HandleFunc("GET /admin/metrics", apiCfg.htmlHandler)
	ServeMux.HandleFunc("POST /admin/reset", apiCfg.resetnumHandler)
	ServeMux.HandleFunc("POST /api/validate_chirp", apiCfg.validateHandler)

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux,
	}

	chripy_server.ListenAndServe()
}
