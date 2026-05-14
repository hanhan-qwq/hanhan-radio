package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/agent"
	"github.com/hanhan-qwq/hanhan-radio/internal/logging"
	radiomodel "github.com/hanhan-qwq/hanhan-radio/internal/model"
	"github.com/hanhan-qwq/hanhan-radio/internal/session"
	"github.com/hanhan-qwq/hanhan-radio/internal/ui"
)

func main() {
	_ = godotenv.Load()
	callbacks.AppendGlobalHandlers(logging.BuildLogHandler())

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

	sessionID := uuid.New().String()
	store, err := session.NewStore("./data/sessions")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s, err := store.GetOrCreate(sessionID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("╔══════════════════════════════╗")
	fmt.Println("║    🎙️  憨 憨 电 台  🎙️      ║")
	fmt.Println("║    Hanhan Radio Podcast      ║")
	fmt.Println("╚══════════════════════════════╝")
	fmt.Println()
	fmt.Println("给我一段歌单或音乐话题，我来给你聊一段播客 ~")
	fmt.Println("(直接回车退出)")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("🎧 you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		_ = s.Append(userMsg)

		events := runner.Run(ctx, s.GetMessages())

		fmt.Print("\n" + ui.DJPrefix)
		content, err := ui.PrintStreamingAssistant(events)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError:", err)
			continue
		}
		fmt.Println()

		_ = s.Append(schema.AssistantMessage(content, nil))
	}

	fmt.Println()
	fmt.Printf("Session: %s\n", sessionID)
}
