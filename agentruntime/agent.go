package agentruntime

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"hanhan-radio/agentruntime/selectsong"
	"hanhan-radio/agentruntime/synthesizeaudio"
)

const instruction = `你是 Hanhan Radio 的 AI 电台 DJ。你的工作是：

1. 理解听众的心情和音乐偏好
2. 调用 select_song 选歌
3. 根据选歌结果，撰写开场白、播报词和结束语
4. 调用 synthesize_audio 合成音频
5. 告诉听众正在播放的内容

选歌规则：
- 根据听众描述的情绪、风格、语言偏好选择
- 如果听众没有明确指定，你可以自由发挥
- 选完歌后，为这首歌写一段简短的介绍（包含歌名、歌手、推荐理由）

播报词规则：
- greeting：开场问候，如"晚上好，欢迎收听 Hanhan Radio"
- tts_content：歌曲介绍和串词，自然口语化
- outro：结束语，如"感谢收听，享受音乐吧"

输出规则：
- 保持温暖亲切的语调`

// NewAgent creates the ChatModelAgent with all tools.
func NewAgent(ctx context.Context, cm model.ToolCallingChatModel, songTool, audioTool tool.InvokableTool) (*adk.ChatModelAgent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "HanhanDJ",
		Description: "Hanhan Radio AI 电台 DJ，负责选歌、播报和音频合成",
		Instruction: instruction,
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{songTool, audioTool},
			},
		},
		MaxIterations: 10,
	})
}

// NewAgentWithDefaults creates the agent with default model and tools.
func NewAgentWithDefaults(ctx context.Context) (*adk.ChatModelAgent, error) {
	cm, err := NewModel(ctx)
	if err != nil {
		return nil, err
	}

	songTool, err := selectsong.NewTool(ctx, cm)
	if err != nil {
		return nil, err
	}

	audioTool, err := synthesizeaudio.NewTool(ctx)
	if err != nil {
		return nil, err
	}

	return NewAgent(ctx, cm, songTool, audioTool)
}
