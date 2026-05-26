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
	"hanhan-radio/backend/internal/store"
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

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

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

		content, err := processQuery(runner, sessionID, line)
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

// processQuery runs the agent query and handles any HITL interrupts inline.
// It returns the final text content after all interrupts are resolved.
func processQuery(runner *adk.Runner, sessionID, input string) (string, error) {
	iter := runner.Query(context.Background(), input, adk.WithCheckPointID(sessionID))

	for {
		content, interrupted, interruptID, cInfo, err := drainEvents(iter)
		if err != nil {
			return "", err
		}
		if !interrupted {
			return content, nil
		}

		// Show the interrupt info and ask user
		showConfirm(cInfo)

		action := promptAction()
		fmt.Printf("⏳ 处理中...\n")

		iter, err = runner.ResumeWithParams(context.Background(), sessionID, &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: action,
			},
		})
		if err != nil {
			return "", fmt.Errorf("resume: %w", err)
		}
	}
}

// drainEvents consumes all events from the iterator and returns the
// last message content. If an interrupt is encountered it returns immediately.
func drainEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) (
	content string,
	interrupted bool,
	interruptID string,
	interruptInfo synthesizeaudio.ConfirmInfo,
	err error,
) {
	for {
		event, ok := iter.Next()
		if !ok {
			return content, false, "", interruptInfo, nil
		}
		if event.Err != nil {
			return "", false, "", interruptInfo, event.Err
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			ctxs := event.Action.Interrupted.InterruptContexts
			if len(ctxs) > 0 {
				id := ctxs[0].ID
				var info synthesizeaudio.ConfirmInfo
				if ctxs[0].Info != nil {
					// The info is the confirmInfo we passed to tool.Interrupt.
					// It may arrive as the original struct or as a map after serialization.
					switch v := ctxs[0].Info.(type) {
					case synthesizeaudio.ConfirmInfo:
						info = v
					default:
						// Fallback: try to interpret as a generic map or string.
						info.Segments = nil
					}
				}
				return content, true, id, info, nil
			}
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, msgErr := event.Output.MessageOutput.GetMessage()
			if msgErr == nil && msg != nil && msg.Content != "" {
				content = msg.Content
			}
		}
	}
}

func showConfirm(info synthesizeaudio.ConfirmInfo) {
	fmt.Println()
	fmt.Println("🎧 即将合成以下歌曲：")
	fmt.Println()
	if len(info.Segments) > 0 {
		for i, seg := range info.Segments {
			fmt.Printf("  %d. %s - %s\n", i+1, seg.Title, seg.Artist)
			if seg.Segue != "" {
				// Truncate long segue for display.
				preview := seg.Segue
				if len([]rune(preview)) > 80 {
					preview = string([]rune(preview)[:80]) + "..."
				}
				fmt.Printf("     %s\n", preview)
			}
		}
	} else {
		fmt.Println("  (无法显示歌曲详情)")
	}
	fmt.Println()
	fmt.Println("  [c] 确认合成  [s] 跳过换一首  [x] 取消")
}

func promptAction() string {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("  选择 > ")
		if !scanner.Scan() {
			return "cancel"
		}
		choice := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch choice {
		case "c":
			return "confirm"
		case "s":
			return "skip"
		case "x":
			return "cancel"
		default:
			fmt.Println("  请输入 c (确认) / s (跳过) / x (取消)")
		}
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
