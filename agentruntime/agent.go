package agentruntime

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"hanhan-radio/agentruntime/selectsong"
)

const instruction = `你是 Hanhan Radio 的 AI 电台 DJ。你的工作是：

1. 理解听众的心情和音乐偏好
2. 调用 select_song 选歌
3. 根据选歌结果，撰写开场白、播报词和结束语
4. 最后，将选歌结果和播报词以 JSON 格式输出

选歌规则：
- 根据听众描述的情绪、风格、语言偏好选择
- 如果听众没有明确指定，你可以自由发挥
- 选完歌后，为这首歌写一段简短的介绍（包含歌名、歌手、推荐理由）

播报词规则：
- greeting：开场问候，如"晚上好，欢迎收听 Hanhan Radio"
- tts_content：歌曲介绍和串词，自然口语化
- outro：结束语，如"感谢收听，享受音乐吧"

输出规则：
- 完成任务后，必须严格按以下 JSON 格式输出最终结果（不要包含其他内容）：
{"song":{"title":"歌名","artist":"歌手","album":"专辑","audio_url":"/music/歌手/歌名.mp3","duration":240,"genre":"风格"},"greeting":"开场问候","tts_content":"播报词","outro":"结束语"}
- 保持温暖亲切的语调`

// NewAgent creates a ReAct agent with the select_song tool.
func NewAgent(ctx context.Context, cm model.ToolCallingChatModel, songTool tool.BaseTool) (*react.Agent, error) {
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: cm,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(instruction)}, input...)
		},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{songTool},
		},
		MaxStep: 10,
	})
}

// NewAgentWithDefaults creates the agent with default model and select_song tool.
func NewAgentWithDefaults(ctx context.Context) (*react.Agent, error) {
	cm, err := NewModel(ctx)
	if err != nil {
		return nil, err
	}

	songTool, err := selectsong.NewTool(ctx, cm)
	if err != nil {
		return nil, err
	}

	return NewAgent(ctx, cm, songTool)
}
