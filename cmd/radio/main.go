package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino/callbacks"

	"github.com/hanhan-qwq/hanhan-radio/internal/agent"
	"github.com/hanhan-qwq/hanhan-radio/internal/logging"
	radiomodel "github.com/hanhan-qwq/hanhan-radio/internal/model"
	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

func main() {
	_ = godotenv.Load()
	callbacks.AppendGlobalHandlers(logging.BuildLogHandler())

	ctx := context.Background()
	cm := radiomodel.NewArkModel()

	graph, err := agent.BuildGraph(ctx, cm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store, err := session.NewStore("./data/sessions")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sess, _ := store.GetOrCreate("debug")
	host := agent.NewRadioHost(graph, sess)
	defer host.Close()

	fmt.Println("🎙️  憨憨电台 (Graph Pipeline)")
	fmt.Println("   memory_load → react_dj → text_preprocess → tts → audio → memory_save")
	fmt.Println()

	events := host.Start()

	for evt := range events {
		switch evt.Type {
		case "text":
			fmt.Print(evt.Data)
		case "done":
			fmt.Println()
		case "error":
			fmt.Fprintf(os.Stderr, "\nError: %s\n", evt.Data)
			return
		}
	}
}
