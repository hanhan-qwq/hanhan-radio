package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"hanhan-radio/agentruntime"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	ctx := context.Background()

	graph, err := agentruntime.NewGraph(ctx)
	if err != nil {
		log.Fatalf("create graph: %v", err)
	}

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

		output, err := graph.Invoke(ctx, input)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			fmt.Println("---")
			continue
		}
		fmt.Printf("音频已生成: %s (时长: %d秒)\n", output.AudioFile, output.Duration)
		fmt.Println("---")
	}
}
