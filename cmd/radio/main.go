package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino-ext/components/model/ark"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/radio"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "用法: go run ./cmd/radio <歌单文件路径>\n")
		os.Exit(1)
	}
	playlistPath := os.Args[1]

	cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL"),
		Thinking: &arkModel.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建模型失败: %v\n", err)
		os.Exit(1)
	}

	tracksJSON := playlistPath + ".json"

	// if tracks.json doesn't exist, tag and generate it
	if _, err := os.Stat(tracksJSON); os.IsNotExist(err) {
		raw, err := playlist.Parse(playlistPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析歌单失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("正在为 %d 首歌打标...\n", len(raw))
		_, err = playlist.TagAndStore(context.Background(), cm, raw, tracksJSON)
		if err != nil {
			fmt.Fprintf(os.Stderr, "打标失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("打标完成")
	}

	r, err := radio.New(radio.Config{
		ChatModel:  cm,
		TracksJSON: tracksJSON,
		PromptsDir: "agentruntime/prompts",
		Interval:   10 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// subscribe to output
	ch := r.Subscribe()
	go func() {
		for text := range ch {
			fmt.Print(text)
		}
	}()

	// stdin reader for user requests
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// format: 歌名 歌手 [状态]
			parts := strings.SplitN(line, " ", 3)
			var t playlist.Track
			var state string
			switch len(parts) {
			case 1:
				t = playlist.Track{Song: parts[0]}
			case 2:
				t = playlist.Track{Song: parts[0], Artist: parts[1]}
			default:
				t = playlist.Track{Song: parts[0], Artist: parts[1]}
				state = parts[2]
			}
			r.Request(t, state)
		}
	}()

	if err := r.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
