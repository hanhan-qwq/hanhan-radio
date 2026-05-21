package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/joho/godotenv"

	_ "hanhan-radio/backend/api"

	"hanhan-radio/agentruntime"
	pkglog "hanhan-radio/agentruntime/log"
	"hanhan-radio/backend/internal/handler"
	"hanhan-radio/backend/internal/manager"
	"hanhan-radio/backend/internal/memory"
	"hanhan-radio/backend/internal/router"
)

// @title           Hanhan Radio API
// @description     AI 电台 DJ — 提交文本，异步生成电台音频
// @version         1.0.0
// @contact.name    Hanhan Radio Team

// @host      localhost:8080
// @BasePath  /api/v1

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	pkglog.InitCallbacks()
	defer pkglog.Sync()

	ctx := context.Background()

	// Initialize SQLite memory store.
	memStore, err := memory.Open("data/hanhan_memory.db")
	if err != nil {
		log.Fatalf("open memory store: %v", err)
	}

	// Create recall_memory tool from the SQLite store.
	recallTool := memory.NewRecallMemoryTool(memStore)

	agent, err := agentruntime.NewAgentWithDefaults(ctx, recallTool)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	store := manager.NewStore()
	epManager := manager.New(runner, store, memStore)
	h := handler.New(epManager, store)
	r := router.New(h)

	pkglog.L().Infow("server_start", "addr", ":8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server: %v", err)
	}
}
