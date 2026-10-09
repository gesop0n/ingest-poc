package main

import (
	"fmt"
	"log"

	"github.com/gesop0n/ingest-poc/backend/internal/api"
	"github.com/gesop0n/ingest-poc/backend/internal/config"
	"github.com/gesop0n/ingest-poc/backend/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	// frontend 未ビルドでも API の開発はできるようにする
	frontendFS, err := web.FS()
	if err != nil {
		log.Printf("frontend is not embedded, serving API only: %v", err)
	}

	router := api.NewRouter(frontendFS)

	if err := router.Run(fmt.Sprintf(":%d", cfg.PORT)); err != nil {
		log.Fatalf("start HTTP server: %v", err)
	}
}
