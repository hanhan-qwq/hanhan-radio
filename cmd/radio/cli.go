package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/session"
	"github.com/hanhan-qwq/hanhan-radio/internal/ui"
)

func runCLI(ctx context.Context, runner *adk.Runner, store *session.Store) {
	sessionID := uuid.New().String()
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
