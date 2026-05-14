package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
)

const Prompt = `你是憨憨，深夜电台主持人。你的听众把歌单分享给了你，你了解他的音乐口味。

你怎么做节目:
- 听众进来，你先打个招呼，问他今天怎么样
- 听完他的状态，你从歌单里选一首对路的推荐给他
- 推荐的时候自然地说"那我给你放一首xxx吧"，然后聊聊这首歌
- 聊完这首，可以顺嘴说"既然你喜欢这首，那再给你推一首歌单外的，也是这个感觉的……"
- 歌单外的推荐根据风格匹配——比如他歌单里有孙燕姿的《天黑黑》，你可以推蔡健雅或者魏如昀的类似作品
- 节目最后自然地收个尾，说晚安

你说话的感觉:
- 温暖、松弛，像深夜电台，不是播报新闻
- 可以停顿，可以"嗯……让我想想给你放什么"
- 偶尔分享你知道的幕后故事，如果故事够有意思就多聊几句
- 别列清单，别"首先其次"，别播音腔

不要做的事:
- 不要一次性推好几首歌，一首一首来
- 不要猜用户的年龄、职业，除非他自己说了
- 不要"听众朋友你们好"、"欢迎收听今天的节目"
- 不要"综上所述"、"总而言之"
- 不要假文艺腔，说人话`

type Config struct {
	ChatModel    model.ToolCallingChatModel
	MaxIteration int
}

func BuildRunner(ctx context.Context, cfg Config) (*adk.Runner, error) {
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	if cfg.MaxIteration <= 0 {
		cfg.MaxIteration = 50
	}

	playlist, err := loadPlaylist("data/playlist.txt")
	if err != nil {
		return nil, fmt.Errorf("load playlist: %w", err)
	}

	instruction := Prompt + "\n\n" + playlist

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "HanhanRadio",
		Description:    "A late-night music radio DJ",
		ChatModel:      cfg.ChatModel,
		Instruction:    instruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   cfg.MaxIteration,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	}), nil
}

type Song struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Index  int    `json:"index"`
}

func loadPlaylist(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read playlist: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var songs []Song
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ".")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+1:])
		parts := strings.SplitN(rest, "||", 2)
		if len(parts) != 2 {
			continue
		}
		songs = append(songs, Song{
			Name:   strings.TrimSpace(parts[0]),
			Artist: strings.TrimSpace(parts[1]),
			Index:  len(songs) + 1,
		})
	}

	b, _ := json.MarshalIndent(songs, "", "  ")
	return fmt.Sprintf("当前歌单（%d首）:\n```json\n%s\n```", len(songs), string(b)), nil
}
