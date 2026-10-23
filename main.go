package main

import (
	"net/http"
)

func main() {
	ServeMux_handler := http.NewServeMux()
	dir := http.Dir(".")
	ServeMux_handler.Handle("/", http.FileServer(dir))

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux_handler,
	}

	chripy_server.ListenAndServe()
}
