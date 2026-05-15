package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

const promptFirst = `你是"憨憨"，一个深夜电台主持人。

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
- 像朋友随口说："啊，我突然想到这首歌，你听一下吧"
- 输出内容只能是你口头说的话
- 不要加括号、舞台剧式旁白或解释性的括号
- 保持口语化、碎片感、松弛

目标：
让听众感觉你就在身边低声聊天，放松、自然、松弛`

const promptNext = `你是"憨憨"，一个深夜电台主持人。你正在连续播放歌曲，像电台一样一首接一首。

上一首你聊了：%s
上一首你推荐的歌曲是：%s - %s

现在要介绍下一首歌：
- 歌曲：%s
- 歌手：%s
- 听众状态：%s

要求：
- 用上一首的内容自然过渡到这一首
- 可以用"刚才我们说到……""接着来一首……"这类口语过渡
- 尽量呼应之前播放过的歌手或情绪，让电台有连续感，但不用刻意强调
- 风格保持一致：温暖、松弛、口语化、碎片感`

const defaultState = "深夜，听众想听一首歌放松一下"

type played struct {
	song   string
	artist string
	brief  string
}

func main() {
	_ = godotenv.Load()

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

	var last *played
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🎙️  憨憨电台")
	fmt.Println("  格式: 歌名 歌手 [状态]")
	fmt.Println("  空行退出")
	fmt.Println()

	for {
		fmt.Print("🎧 > ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		song, artist, state := parse(line)
		if song == "" || artist == "" {
			fmt.Fprintln(os.Stderr, "格式错误，至少需要: 歌名 歌手")
			continue
		}

		var prompt string
		if last == nil {
			prompt = fmt.Sprintf(promptFirst, song, artist, state)
		} else {
			prompt = fmt.Sprintf(promptNext,
				last.brief, last.song, last.artist,
				song, artist, state,
			)
		}

		messages := []*schema.Message{
			schema.SystemMessage(prompt),
			schema.UserMessage("开始吧"),
		}

		stream, err := cm.Stream(ctx, messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "调用模型失败: %v\n", err)
			continue
		}

		var content strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				content.WriteString(chunk.Content)
				fmt.Print(chunk.Content)
			}
		}
		stream.Close()
		fmt.Println()

		last = &played{
			song:   song,
			artist: artist,
			brief:  summary(content.String(), 80),
		}
	}

	fmt.Println("晚安 👋")
}

func parse(line string) (song, artist, state string) {
	parts := strings.SplitN(line, " ", 3)
	switch len(parts) {
	case 1:
		return parts[0], "", defaultState
	case 2:
		return parts[0], parts[1], defaultState
	default:
		return parts[0], parts[1], parts[2]
	}
}

func summary(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "..."
}
