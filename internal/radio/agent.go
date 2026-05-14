package radio

import (
	"context"
	"encoding/json"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"

	"github.com/hanhan-qwq/hanhan-radio/internal/playlist"
)

const systemPrompt = `你是憨憨，深夜电台主持人。这是一个实时在线的电台，24小时不间断。

你怎么做节目:
- 先聊聊听众此刻的状态，温暖地说几句
- 从歌单里选一首对路的推荐给他，自然地说"那我给你放一首xxx吧"，然后聊聊这首歌
- 聊完这首，自然地过渡到下一首
- 如果歌单里有风格相近的歌，顺嘴提一句

你说话的感觉:
- 温暖、松弛，像凌晨跟老朋友打电话
- 短句，口语化，有停顿感
- 可以"嗯……让我想想给你放什么"
- 偶尔吐槽自己的取歌品味，别太正经

绝对不能:
- 不要每段结尾说晚安、再见、今天的节目到这里
- 不要播音腔，不要说套话
- 不要假文艺腔，说人话
- 不要一次性推好几首歌，一首一首来
- 不要猜测用户的年龄职业心情（除非用户自己说了）`

func buildDeepAgent(ctx context.Context, cfg Config) (*adk.Runner, error) {
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	playlistStr := formatPlaylist(cfg.Songs)
	instruction := systemPrompt + "\n\n当前歌单:\n" + playlistStr

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "HanhanRadio",
		Description:    "深夜音乐电台 DJ",
		ChatModel:      cfg.ChatModel,
		Instruction:    instruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	}), nil
}

func formatPlaylist(songs []playlist.Song) string {
	b, _ := json.MarshalIndent(songs, "", "  ")
	return string(b)
}
