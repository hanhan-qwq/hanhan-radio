package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/joho/godotenv"

	"hanhan-radio/agentruntime"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	ctx := context.Background()

	agent, err := agentruntime.NewAgentWithDefaults(ctx)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	fmt.Println("Hanhan Radio — AI 电台 DJ")
	fmt.Println("输入你想听的音乐（如 '想听点轻松的'），输入 exit 退出")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "exit" || input == "quit" {
			fmt.Println("再见～")
			break
		}
		if input == "" {
			continue
		}

		iter := runner.Query(ctx, input)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
				continue
			}
			if event.Output != nil && event.Output.MessageOutput != nil {
				mv := event.Output.MessageOutput
				if mv.IsStreaming {
					for {
						msg, err := mv.MessageStream.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							log.Printf("stream error: %v", err)
							break
						}
						fmt.Print(msg.Content)
					}
					fmt.Println()
				} else {
					msg, _ := mv.GetMessage()
					if msg != nil && msg.Content != "" {
						fmt.Printf("[%s] %s\n", mv.Role, msg.Content)
					}
				}
			}
		}
		fmt.Println("---")
	}
}
