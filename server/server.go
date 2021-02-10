package server

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// RunBackend server
func RunBackend(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from the proxy backend in port %s! %s", port, time.Now())
	})

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
