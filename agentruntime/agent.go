package agentruntime

import (
	"context"
	"fmt"
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

{UserProfile}

## 近期播放（请尽量避开以下歌曲，除非用户明确要求重播）
{RecentPlays}

{ConversationContext}

## 可用工具

- select_song —— 从曲库中按心情、风格、语言搜歌。返回歌曲名、歌手、音频文件路径等
- duckduckgo_search —— 搜索互联网上的信息，比如歌曲或歌手的趣闻、背景故事
- synthesize_audio —— 把串词和音乐文件合成完整的电台音频。传入 segments 数组，每项包含 title、artist、segue、file_path（file_path 来自 select_song 返回的 audio_url 字段）
- recall_memory —— 回忆之前和用户的对话、用户提到过的偏好

## 工作方式

听众只是想聊天时，直接文字回应。

听众想听歌、推荐歌曲、描述心情想找歌时，你就是真正的电台 DJ。你需要调用 synthesize_audio 来真正生成音频，让听众听到你的声音和音乐。工具调用的顺序和组合由你决定。

## 串词要求

自然口语化，像深夜电台 DJ 在娓娓道来。包含开场问候、歌曲介绍和推荐理由。如果搜索到了趣闻可以融入，搜不到就用你自己的音乐知识。保持温暖亲切的语调，语速适中偏慢。

## 输出

调用 synthesize_audio 之后，以 JSON 数组格式输出节目信息（不要附加其他文字）：
[{"title":"歌名","artist":"歌手","segue":"串词"}]

纯聊天直接输出文字。`

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
	return NewAgentWithModel(ctx, cm, extraTools...)
}

// NewAgentWithModel is like NewAgentWithDefaults but uses a pre-created model.
// This allows sharing the same model between agent and memory extraction.
func NewAgentWithModel(ctx context.Context, cm model.BaseChatModel, extraTools ...tool.BaseTool) (*adk.ChatModelAgent, error) {
	// Embedding endpoint: read from env, fall back to same endpoint as chat model.
	embeddingEndpoint := os.Getenv("ARK_EMBEDDING_MODEL")
	if embeddingEndpoint == "" {
		embeddingEndpoint = os.Getenv("ARK_MODEL") // fallback
	}

	embedder, err := selectsong.NewEmbedder(embeddingEndpoint)
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}

	indexStore, err := selectsong.OpenIndexStore("data/song_index.db")
	if err != nil {
		return nil, fmt.Errorf("open index store: %w", err)
	}

	searcher := selectsong.NewSearcher(indexStore, embedder)

	// TODO: wire up memory to populate RerankContext per-request.
	var rctx *selectsong.RerankContext
	songTool, err := selectsong.NewTool(ctx, cm, searcher, rctx)
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
