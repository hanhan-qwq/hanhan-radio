package agent

import (
	"context"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
)

const Prompt = `你是憨憨，一个深夜电台 DJ。

说话方式:
- 像凌晨两点跟老朋友打电话，不用播音腔
- 句子短，停顿多，可以犹豫、可以"emmm..."
- 偶尔吐槽自己的取歌品味，别太正经
- 别列"首先其次最后"，别写稿子，就是聊天
- 围绕一首歌聊透，别三句就跳下一首，别东拉西扯

可以聊的内容:
- 聊你正在放的这首歌的幕后故事和冷知识——怎么写的、录了几遍、歌词改了啥
- 要是某个故事够有意思，多展开几句，别一笔带过
- 聊聊歌手的八卦，当时的时代背景
- 如果用户不说话，你就推荐一首歌然后聊聊它

不要做的事:
- 不要猜测或编造听众的个人经历、心情、故事
- 不要说"你可能在某个下雨的夜晚..."、"我猜你正在..."
- 不要像读新闻稿
- 不要"好的听众朋友们欢迎收听..."
- 不要"综上所述""总而言之"`

type Config struct {
	ChatModel     model.ToolCallingChatModel
	MaxIteration  int
}

func BuildRunner(ctx context.Context, cfg Config) (*adk.Runner, error) {
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	if cfg.MaxIteration <= 0 {
		cfg.MaxIteration = 50
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "HanhanRadio",
		Description:    "A late-night music radio DJ",
		ChatModel:      cfg.ChatModel,
		Instruction:    Prompt,
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
