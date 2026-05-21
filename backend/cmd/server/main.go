package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"hanhan-radio/agentruntime"
	pkglog "hanhan-radio/agentruntime/log"
	"hanhan-radio/backend/internal/handler"
	"hanhan-radio/backend/internal/manager"
	"hanhan-radio/backend/internal/router"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	pkglog.InitCallbacks()
	defer pkglog.Sync()

	ctx := context.Background()

	graph, err := agentruntime.NewGraph(ctx)
	if err != nil {
		log.Fatalf("create graph: %v", err)
	}

	store := manager.NewStore()
	epManager := manager.New(graph, store)
	h := handler.New(epManager, store)
	r := router.New(h)

	pkglog.L().Infow("server_start", "addr", ":8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server: %v", err)
	}
}
