package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino/callbacks"

	"github.com/hanhan-qwq/hanhan-radio/internal/agent"
	"github.com/hanhan-qwq/hanhan-radio/internal/logging"
	radiomodel "github.com/hanhan-qwq/hanhan-radio/internal/model"
	"github.com/hanhan-qwq/hanhan-radio/internal/server"
	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

func main() {
	_ = godotenv.Load()
	callbacks.AppendGlobalHandlers(logging.BuildLogHandler())

	serve := flag.Bool("serve", false, "start HTTP server")
	addr := flag.String("addr", ":8080", "server listen address")
	flag.Parse()

	cm := radiomodel.NewArkModel()

	ctx := context.Background()
	runner, err := agent.BuildRunner(ctx, agent.Config{
		ChatModel:    cm,
		MaxIteration: 50,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store, err := session.NewStore("./data/sessions")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *serve {
		srv := server.New(runner, store)
		if err := srv.Start(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// CLI mode
	runCLI(ctx, runner, store)
}
