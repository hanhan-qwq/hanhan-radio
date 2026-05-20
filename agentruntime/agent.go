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

串词规则：
- 串词（segue）包含对这首歌的介绍和过渡语，自然口语化
- 如果是第一首歌，串词可以包含开场问候
- 如果是最后一首歌，串词可以包含结束语

输出规则：
- 完成任务后，必须严格按以下 JSON 数组格式输出最终结果（不要包含其他内容）：
[{"title":"歌名","artist":"歌手","segue":"串词"}]
- 数组中每个元素包含一首歌的 title（歌名）、artist（歌手）和 segue（串词）
- 串词要自然口语化，可以包含开场白、歌曲介绍和结束语
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
