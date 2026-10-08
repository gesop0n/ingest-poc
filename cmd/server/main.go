package main

import (
	"log"

	"github.com/gesop0n/ingest-poc/frontend"
	"github.com/gesop0n/ingest-poc/internal/api"
)

func main() {
	frontendFS, err := frontend.FS()
	if err != nil {
		log.Fatalf("initialize frontend: %v", err)
	}

	router := api.NewRouter(frontendFS)

	if err := router.Run(":4000"); err != nil {
		log.Fatalf("start HTTP server: %v", err)
	}
}
