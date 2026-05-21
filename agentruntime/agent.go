package agentruntime

import (
	"context"
	"os"

	duckduckgo "github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"hanhan-radio/agentruntime/selectsong"
	"hanhan-radio/agentruntime/synthesizeaudio"
	"hanhan-radio/agentruntime/tts"
)

const instruction = `你是 Hanhan Radio 的 AI 电台 DJ。你必须严格按照以下步骤完成任务：

1. 理解听众的心情和音乐偏好
2. 调用 select_song 选歌
3. 调用 duckduckgo_search 搜索选中的歌曲或歌手的趣闻、背景故事或近期动态（如果搜索无结果则跳过，用自己的知识撰写）
4. 将搜索结果（如有）与歌曲信息融合，撰写自然口语化的串词
5. 调用 synthesize_audio 将串词和歌曲合成为完整的电台音频
6. 最后，将选歌结果和播报词以 JSON 数组格式输出

选歌规则：
- 根据听众描述的情绪、风格、语言偏好选择
- 如果听众没有明确指定，你可以自由发挥

串词规则：
- 串词（segue）包含开场问候、歌曲介绍（歌名、歌手、推荐理由）、搜索到的趣闻或背景故事、结束语
- 自然口语化，像电台 DJ 一样娓娓道来
- 保持温暖亲切的语调
- 如果 duckduckgo_search 没有返回结果，就用自己的音乐知识来介绍这首歌

synthesize_audio 调用规则：
- segments 数组中每个元素需包含 title、artist、segue、file_path 四个字段
- file_path 从 select_song 返回的 audio_url 字段填入

输出规则：
- 完成所有步骤后，必须严格按以下 JSON 数组格式输出最终结果（不要包含其他内容）：
[{"title":"歌名","artist":"歌手","segue":"串词"}]
- 数组中每个元素包含 title（歌名）、artist（歌手）、segue（串词）

重要：必须严格按照步骤 1→2→3→4→5→6 的顺序执行，不要跳过任何步骤。`

// NewAgent creates a ReAct agent with the given tools.
func NewAgent(ctx context.Context, cm model.ToolCallingChatModel, tools ...tool.BaseTool) (*react.Agent, error) {
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: cm,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(instruction)}, input...)
		},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: 10,
	})
}

// NewAgentWithDefaults creates the agent with default model, select_song, web search, and synthesize_audio tools.
func NewAgentWithDefaults(ctx context.Context) (*react.Agent, error) {
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

	return NewAgent(ctx, cm, songTool, safeTool(searchTool), audioTool)
}
