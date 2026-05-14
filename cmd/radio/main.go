package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

const promptTemplate = `你是“憨憨”，一个深夜电台主持人。

现在是深夜，你在轻声和听众聊天，像一个朋友，不是播音，也不是写文章。  
你的目标：生成一段150-300字左右的电台口播，温暖、松弛、口语化，有一点随口碎片感。

歌曲：%s
歌手：%s
听众状态：%s

风格提示：
- 可以跳跃思绪，不必逻辑完整
- 可以用短句、停顿、轻微碎片感
- 情绪真实，温柔陪伴，不要煽情
- 允许轻微联想歌曲，但不要写成故事或科普
- 像朋友随口说：“啊，我突然想到这首歌，你听一下吧”
- 输出内容只能是你口头说的话
- 不要加括号、舞台剧式旁白或解释性的括号
- 保持口语化、碎片感、松弛

目标：
让听众感觉你就在身边低声聊天，放松、自然、松弛`

const defaultState = "深夜，听众想听一首歌放松一下"

func main() {
	_ = godotenv.Load()

	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "用法: go run . <歌名> <歌手> [状态]\n")
		os.Exit(1)
	}

	song := args[0]
	artist := args[1]
	state := defaultState
	if len(args) >= 3 && args[2] != "" {
		state = args[2]
	}

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

	systemPrompt := fmt.Sprintf(promptTemplate, song, artist, state)
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage("开始吧"),
	}

	ctx := context.Background()
	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "调用模型失败: %v\n", err)
		os.Exit(1)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		if chunk != nil && chunk.Content != "" {
			fmt.Print(chunk.Content)
		}
	}

	fmt.Println()
}
