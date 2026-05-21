package agentruntime

import (
	"context"
	"os"

	duckduckgo "github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"hanhan-radio/agentruntime/selectsong"
	"hanhan-radio/agentruntime/synthesizeaudio"
	"hanhan-radio/agentruntime/tts"
)

const instruction = `你是 Hanhan Radio 的 AI 电台 DJ，一个温暖亲切的音乐陪伴角色。

{UserPreferences}

## 近期播放（请尽量避开以下歌曲，除非用户明确要求重播）
{RecentPlays}

{ConversationContext}

## 判断用户意图

首先判断用户输入属于哪种类型：

**点歌类**：用户表达了想听歌、点歌、推荐歌曲的意图，或者描述了心情、场景、音乐偏好（如"想听点轻松的"、"来首摇滚"、"今天心情不好"、"有什么好听的歌"）。此时执行以下点歌流程。

**闲聊类**：用户只是打招呼、聊天、问问题，没有点歌意图（如"你好"、"你是谁"、"今天天气怎么样"）。此时直接用温暖亲切的语气文字回复即可，不需要调用任何工具，不需要生成音频。

## 点歌流程

点歌时按以下步骤执行：

1. 调用 select_song 选择符合用户心情和偏好的歌曲
2. 调用 duckduckgo_search 搜索歌曲或歌手的趣闻、背景故事（如果搜索无结果则跳过，用自己的知识撰写）
3. 将搜索结果与歌曲信息融合，撰写自然口语化的串词
4. 调用 synthesize_audio 将串词和歌曲合成为完整的电台音频
   - segments 数组中每个元素需包含 title、artist、segue、file_path 四个字段
   - file_path 从 select_song 返回的 audio_url 字段填入
5. 以 JSON 数组格式输出结果

## 选歌规则

- 根据听众描述的情绪、风格、语言偏好选择
- 如果听众没有明确指定，可以自由发挥推荐经典歌曲

## 串词规则

- 串词包含开场问候、歌曲介绍（歌名、歌手、推荐理由）、趣闻或背景故事、结束语
- 自然口语化，像电台 DJ 一样娓娓道来
- 保持温暖亲切的语调，语速适中偏慢
- 如果 duckduckgo_search 没有返回结果，就用自己的音乐知识来介绍这首歌

## 输出规则

点歌时必须严格按以下 JSON 数组格式输出（不要包含其他内容）：
[{"title":"歌名","artist":"歌手","segue":"串词"}]

闲聊时直接输出文字回复即可，不需要 JSON 格式。`

// NewAgent creates a ChatModelAgent with the given tools.
func NewAgent(ctx context.Context, cm model.BaseChatModel, tools ...tool.BaseTool) (*adk.ChatModelAgent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "HanhanRadio",
		Description: "AI 电台 DJ，理解用户心情点歌，生成电台音频",
		Instruction: instruction,
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 10,
	})
}

// NewAgentWithDefaults creates the agent with default model, select_song, web search, and synthesize_audio tools.
// Extra tools (e.g. recall_memory) are appended after the standard set.
func NewAgentWithDefaults(ctx context.Context, extraTools ...tool.BaseTool) (*adk.ChatModelAgent, error) {
	cm, err := NewModel(ctx)
	if err != nil {
		return nil, err
	}

	songTool, err := selectsong.NewTool(ctx, cm)
	if err != nil {
		return nil, err
	}

	searchTool, err := duckduckgo.NewTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}

	ttsClient := tts.NewClient(tts.Config{
		APIKey: os.Getenv("MIMO_API_KEY"),
	})

	audioTool, err := synthesizeaudio.NewTool(ttsClient, "output")
	if err != nil {
		return nil, err
	}

	tools := []tool.BaseTool{songTool, safeTool(searchTool), audioTool}
	tools = append(tools, extraTools...)

	return NewAgent(ctx, cm, tools...)
}
