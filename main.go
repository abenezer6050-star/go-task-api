package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           NewServer(NewStore()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("task API listening on http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
