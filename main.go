package main

import (
	"fmt"
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

func (cfg *apiConfig) resetnumHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Swap(0)
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

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux,
	}

	chripy_server.ListenAndServe()
}
