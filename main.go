package main

import (
	"net/http"
)

func main() {
	ServeMux_handler := http.NewServeMux()

	chripy_server := http.Server{
		Addr:    ":8080",
		Handler: ServeMux_handler,
	}

	chripy_server.ListenAndServe()
}
