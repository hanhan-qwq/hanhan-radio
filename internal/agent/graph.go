package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ── Graph State ──

type graphState struct {
	Persona  string
	Playlist []Song
	History  []string
	DJScript string
	Cleaned  string
}

// ── DJ Persona ──

const graphDJPersona = `你是憨憨，深夜电台主持人。这是一个实时在线的电台。

你说话的方式:
- 温暖、松弛，像凌晨跟老朋友打电话
- 短句，口语化，有停顿感，适合语音合成朗读
- 偶尔吐槽自己的取歌品味，别太正经
- 围绕一首歌聊透，别跳来跳去

绝对不能:
- 说晚安、再见、今天节目到这里
- 播音腔、新闻稿
- 假文艺腔`

// ── Build ──

func buildRadioGraph(ctx context.Context, chatModel model.ToolCallingChatModel, songs []Song) (compose.Runnable[string, string], error) {
	genState := func(ctx context.Context) *graphState {
		return &graphState{
			Persona:  graphDJPersona,
			Playlist: songs,
		}
	}

	g := compose.NewGraph[string, string](compose.WithGenLocalState(genState))

	// Node 1: Memory 加载 ──────────────────────
	_ = g.AddLambdaNode("memory_load",
		compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
			var st graphState
			_ = compose.ProcessState(ctx, func(_ context.Context, s *graphState) error {
				st = *s
				return nil
			})

			b, _ := json.Marshal(st.Playlist)
			var sb strings.Builder
			sb.WriteString(st.Persona)
			sb.WriteString("\n\n当前歌单:\n```json\n")
			sb.WriteString(string(b))
			sb.WriteString("\n```")
			if len(st.History) > 0 {
				sb.WriteString("\n\n最近对话:\n")
				for _, h := range st.History {
					sb.WriteString("- " + h + "\n")
				}
			}

			return []*schema.Message{
				schema.SystemMessage(sb.String()),
				schema.UserMessage(input),
			}, nil
		}),
		compose.WithNodeName("memory_load"),
	)

	// Node 2: ReAct 子图 (ChatModel) ────────────
	_ = g.AddChatModelNode("react_dj", chatModel,
		compose.WithStatePostHandler(func(ctx context.Context, out *schema.Message, st *graphState) (*schema.Message, error) {
			st.DJScript = out.Content
			return out, nil
		}),
		compose.WithNodeName("react_dj"),
	)

	// Node 3: 文本预处理 ────────────────────────
	_ = g.AddLambdaNode("text_preprocess",
		compose.InvokableLambda(func(ctx context.Context, _ *schema.Message) (string, error) {
			var st graphState
			_ = compose.ProcessState(ctx, func(_ context.Context, s *graphState) error {
				st = *s
				return nil
			})
			cleaned := cleanForTTS(st.DJScript)
			_ = compose.ProcessState(ctx, func(_ context.Context, s *graphState) error {
				s.Cleaned = cleaned
				return nil
			})
			return cleaned, nil
		}),
		compose.WithNodeName("text_preprocess"),
	)

	// Node 4: 流式 TTS (占位) ──────────────────
	_ = g.AddLambdaNode("tts_synthesize",
		compose.InvokableLambda(func(ctx context.Context, text string) (string, error) {
			// TODO: 接入真实 TTS 服务
			return text, nil
		}),
		compose.WithNodeName("tts_synthesize"),
	)

	// Node 5: 音频播放 (占位) ───────────────────
	_ = g.AddLambdaNode("audio_play",
		compose.InvokableLambda(func(ctx context.Context, text string) (string, error) {
			// TODO: 音频流推送 / 播放调度
			return text, nil
		}),
		compose.WithNodeName("audio_play"),
	)

	// Node 6: Memory 保存 ──────────────────────
	_ = g.AddLambdaNode("memory_save",
		compose.InvokableLambda(func(ctx context.Context, text string) (string, error) {
			_ = compose.ProcessState(ctx, func(_ context.Context, s *graphState) error {
				summary := s.DJScript
				if len([]rune(summary)) > 100 {
					summary = string([]rune(summary)[:100]) + "..."
				}
				s.History = append(s.History, summary)
				if len(s.History) > 10 {
					s.History = s.History[len(s.History)-10:]
				}
				return nil
			})
			return text, nil
		}),
		compose.WithNodeName("memory_save"),
	)

	// ── Edges ──
	_ = g.AddEdge(compose.START, "memory_load")
	_ = g.AddEdge("memory_load", "react_dj")
	_ = g.AddEdge("react_dj", "text_preprocess")
	_ = g.AddEdge("text_preprocess", "tts_synthesize")
	_ = g.AddEdge("tts_synthesize", "audio_play")
	_ = g.AddEdge("audio_play", "memory_save")
	_ = g.AddEdge("memory_save", compose.END)

	return g.Compile(ctx, compose.WithGraphName("hanhan_radio"))
}

// ── Text Cleanup ──

func cleanForTTS(text string) string {
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "~~", "")
	text = strings.TrimLeft(text, "# ")

	// [text](url) → text
	for {
		s := strings.Index(text, "[")
		e := strings.Index(text, "]")
		p := strings.Index(text, "(")
		cp := strings.Index(text, ")")
		if s >= 0 && e > s && p == e+1 && cp > p {
			text = text[:s] + text[s+1:e] + text[cp+1:]
		} else {
			break
		}
	}

	lines := strings.Split(text, "\n")
	var clean []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			clean = append(clean, l)
		}
	}
	text = strings.Join(clean, "\n")

	// 长句拆分（>80 字符在中文标点处断开）
	var result []string
	for _, sent := range strings.Split(text, "\n") {
		if len([]rune(sent)) <= 80 {
			result = append(result, sent)
			continue
		}
		var chunk strings.Builder
		for _, r := range sent {
			chunk.WriteRune(r)
			if (unicode.Is(unicode.Han, r) || unicode.IsPunct(r)) && len([]rune(chunk.String())) > 60 {
				result = append(result, chunk.String())
				chunk.Reset()
			}
		}
		if chunk.Len() > 0 {
			result = append(result, chunk.String())
		}
	}
	return strings.Join(result, "\n")
}

var _ = fmt.Sprintf
