package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/selector"
)

type Host struct {
	cm       model.ToolCallingChatModel
	tools    []tool.BaseTool
	firstTpl string
	nextTpl  string
}

func New(cm model.ToolCallingChatModel, promptsDir string, tools []tool.BaseTool) (*Host, error) {
	first, err := os.ReadFile(filepath.Join(promptsDir, "first.txt"))
	if err != nil {
		return nil, fmt.Errorf("read first.txt: %w", err)
	}
	next, err := os.ReadFile(filepath.Join(promptsDir, "next.txt"))
	if err != nil {
		return nil, fmt.Errorf("read next.txt: %w", err)
	}
	return &Host{cm: cm, tools: tools, firstTpl: string(first), nextTpl: string(next)}, nil
}

// Generate produces a streaming script for the given track.
func (h *Host) Generate(ctx context.Context, track playlist.Track, sel *selector.Result, last *LastPlayed, state, timeInfo, festival string) (*schema.StreamReader[*schema.Message], error) {
	var systemPrompt string
	if last == nil {
		systemPrompt = h.firstTpl
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{song}}", track.Song)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{artist}}", track.Artist)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{reason}}", sel.Reason)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{state}}", state)
		systemPrompt = injectTimeContext(systemPrompt, timeInfo, festival)
	} else {
		systemPrompt = h.nextTpl
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{last_brief}}", last.Brief)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{last_song}}", last.Track.Song)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{last_artist}}", last.Track.Artist)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{song}}", track.Song)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{artist}}", track.Artist)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{reason}}", sel.Reason)
		systemPrompt = strings.ReplaceAll(systemPrompt, "{{state}}", state)
		systemPrompt = injectTimeContext(systemPrompt, timeInfo, festival)
	}

	// build agent with tools
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "HanhanHost",
		Description: "深夜电台主持人生成串场词",
		Model:       h.cm,
		Instruction: systemPrompt,
		MaxIterations: 3,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: h.tools,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// run agent and convert events to stream
	r, w := schema.Pipe[*schema.Message](8)
	go func() {
		defer w.Close()
		events := runner.Run(ctx, []*schema.Message{
			schema.UserMessage("开始吧"),
		})
		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				w.Send(nil, event.Err)
				return
			}
			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			mv := event.Output.MessageOutput
			// skip tool messages and planning messages with tool calls
			if mv.Role == schema.Tool || (mv.Role != schema.Assistant && mv.Role != "") {
				continue
			}
			if mv.IsStreaming {
				mv.MessageStream.SetAutomaticClose()
				for {
					frame, err := mv.MessageStream.Recv()
					if err != nil {
						break
					}
					// skip frames that contain tool calls (planning phase)
					if frame != nil && len(frame.ToolCalls) > 0 {
						continue
					}
					if frame != nil && frame.Content != "" {
						w.Send(frame, nil)
					}
				}
				continue
			}
			// skip non-streaming messages with tool calls
			if mv.Message != nil && len(mv.Message.ToolCalls) == 0 {
				w.Send(mv.Message, nil)
			}
		}
	}()

	return r, nil
}

func injectTimeContext(prompt, timeInfo, festival string) string {
	var sb strings.Builder
	sb.WriteString("\n\n当前时间：")
	sb.WriteString(timeInfo)
	if festival != "" {
		sb.WriteString("，今天是")
		sb.WriteString(festival)
	}
	sb.WriteString("。")
	return prompt + sb.String()
}

// LastPlayed holds the previous track info for continuity.
type LastPlayed struct {
	Track playlist.Track
	Brief string
}

// Summary returns the first n runes of text.
func Summary(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "..."
}
