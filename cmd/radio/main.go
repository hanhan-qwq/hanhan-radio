package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino/callbacks"

	"github.com/hanhan-qwq/hanhan-radio/internal/callback"
	"github.com/hanhan-qwq/hanhan-radio/internal/model"
	"github.com/hanhan-qwq/hanhan-radio/internal/playlist"
	"github.com/hanhan-qwq/hanhan-radio/internal/radio"
	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

func main() {
	_ = godotenv.Load()
	callbacks.AppendGlobalHandlers(callback.Handler())

	ctx := context.Background()
	cm := model.NewArkModel()

	songs, err := playlist.Load("data/playlist.txt")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner, err := radio.BuildRunner(ctx, radio.Config{
		ChatModel: cm,
		Songs:     songs,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store, _ := session.NewStore("./data/sessions")
	sess, _ := store.GetOrCreate("debug")
	host := radio.NewHost(runner, sess)
	defer host.Close()

	fmt.Println("🎙️  憨憨电台")
	fmt.Println("  DeepAgent (ReAct) → 流式输出")
	fmt.Println()

	for evt := range host.Start() {
		switch evt.Type {
		case "text":
			fmt.Print(evt.Data)
		case "tool":
			fmt.Printf("\n  [tool] %s\n", evt.Data)
		case "done":
			fmt.Println()
		case "error":
			fmt.Fprintf(os.Stderr, "\nError: %s\n", evt.Data)
			return
		}
	}
}
