package main

import (
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	ServeMux := http.NewServeMux()

	dir := http.Dir(".")
	wrappedhandler := http.StripPrefix("/app", http.FileServer(dir))
	ServeMux.Handle("/app/", wrappedhandler)

	ServeMux.HandleFunc("/healthz", Handler)

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux,
	}

	chripy_server.ListenAndServe()
}
