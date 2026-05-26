package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/joho/godotenv"

	"hanhan-radio/agentruntime"
	pkglog "hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/synthesizeaudio"
	"hanhan-radio/backend/internal/memory"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}

	pkglog.InitCallbacks()
	defer pkglog.Sync()

	ctx := context.Background()

	cm, err := agentruntime.NewModel(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create model: %v\n", err)
		os.Exit(1)
	}

	memStore, err := memory.Open("data/hanhan_memory.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open memory: %v\n", err)
		os.Exit(1)
	}

	recallTool := memory.NewRecallMemoryTool(memStore)

	agent, err := agentruntime.NewAgentWithModel(ctx, cm, recallTool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create agent: %v\n", err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	sessionID := genSessionID()

	fmt.Println("🎵 Hanhan Radio CLI")
	fmt.Println("输入你的心情或想听的歌，输入 /quit 退出")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/quit" || line == "/q" {
			fmt.Println("再见~")
			break
		}

		start := time.Now()

		// Inject memory context.
		values := memStore.Preprocess(sessionID)
		adk.AddSessionValues(ctx, values)

		iter := runner.Query(ctx, line)

		content, err := consumeOutput(iter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			continue
		}

		content = strings.TrimSpace(content)

		if strings.HasPrefix(content, "[") {
			handleMusic(ctx, content, start)
		} else {
			fmt.Println()
			fmt.Println(content)
			fmt.Printf("\n⏱  %.1fs\n", time.Since(start).Seconds())
		}

		// Post-process: record history, extract facts.
		memStore.Postprocess(sessionID, line, content)
		go func() {
			if err := memStore.ExtractFacts(context.Background(), cm, line, content); err != nil {
				pkglog.L().Warnw("extract_facts_failed", "err", err)
			}
			if err := memStore.ConsolidateFacts(context.Background(), cm); err != nil {
				pkglog.L().Warnw("consolidate_facts_failed", "err", err)
			}
		}()
	}
}

func handleMusic(ctx context.Context, content string, start time.Time) {
	var items []struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
		Segue  string `json:"segue"`
	}
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		fmt.Fprintf(os.Stderr, "❌ parse response: %v\n", err)
		return
	}

	fmt.Println()
	for i, item := range items {
		fmt.Printf("🎵 %d. %s - %s\n", i+1, item.Title, item.Artist)
		if item.Segue != "" {
			fmt.Printf("   %s\n", item.Segue)
		}
	}

	audioPath := filepath.Join("output", "final.mp3")
	if dur, err := synthesizeaudio.ProbeDuration(ctx, audioPath); err == nil {
		fmt.Printf("\n📻 音频: %s (%s)\n", audioPath, formatDuration(dur))
	}
	fmt.Printf("⏱  %.1fs\n", time.Since(start).Seconds())
}

func consumeOutput(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var last string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil && msg != nil && msg.Content != "" {
				last = msg.Content
			}
		}
	}
	if last == "" {
		return "", fmt.Errorf("empty response")
	}
	return last, nil
}

func genSessionID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "s_" + hex.EncodeToString(b)
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
}
