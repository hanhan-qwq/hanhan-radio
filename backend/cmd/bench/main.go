package main

import (
	"context"
	"fmt"
	"os"
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
	_ = godotenv.Load()
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

	sessionID := "bench_session"
	memStore.Preprocess(sessionID)

	queries := []string{
		"想听一首温暖治愈的歌",
	}

	for _, q := range queries {
		fmt.Printf("\n━━━ 查询: %s ━━━\n", q)
		totalStart := time.Now()

		iter := runner.Query(ctx, q, adk.WithCheckPointID(sessionID))

		var (
			lastContent      string
			interruptCount   int
			firstInterruptAt time.Duration
		)

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Printf("  ❌ error: %v\n", event.Err)
				break
			}

			if event.Action != nil && event.Action.Interrupted != nil {
				if interruptCount == 0 {
					firstInterruptAt = time.Since(totalStart)
					fmt.Printf("  ⏱  首次中断出现: %.2fs\n", firstInterruptAt.Seconds())
				}
				interruptCount++

				// Auto-confirm
				ctxs := event.Action.Interrupted.InterruptContexts
				if len(ctxs) > 0 {
					iter, err = runner.ResumeWithParams(ctx, sessionID, &adk.ResumeParams{
						Targets: map[string]any{ctxs[0].ID: "confirm"},
					})
					if err != nil {
						fmt.Printf("  ❌ resume error: %v\n", err)
						break
					}
					fmt.Printf("  ✅ 自动确认 (中断 %d)\n", interruptCount)
				}
				continue
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				msg, msgErr := event.Output.MessageOutput.GetMessage()
				if msgErr == nil && msg != nil && msg.Content != "" {
					lastContent = msg.Content
				}
			}
		}

		totalElapsed := time.Since(totalStart)
		fmt.Printf("  ⏱  总耗时 (含自动确认): %.2fs\n", totalElapsed.Seconds())

		// Show output preview
		if len(lastContent) > 200 {
			lastContent = lastContent[:200] + "..."
		}
		fmt.Printf("  📝 输出: %s\n", lastContent)

		// Check if output file exists
		if dur, err := synthesizeaudio.ProbeDuration(ctx, "output/final.mp3"); err == nil {
			fmt.Printf("  📻 final.mp3 时长: %ds\n", dur)
		}
	}
}
