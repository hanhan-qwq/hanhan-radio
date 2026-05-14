package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const DJPrefix = "🎙️  憨憨> "

// PrintStreamingAssistant consumes events from the runner and prints content as it arrives.
// Returns the full concatenated assistant message.
func PrintStreamingAssistant(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		if mv.Role == schema.Tool {
			content := drainMessage(mv)
			fmt.Printf("\n  [tool] %s\n%s", Truncate(content, 200), DJPrefix)
			continue
		}

		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming {
			mv.MessageStream.SetAutomaticClose()
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil {
					if frame.Content != "" {
						sb.WriteString(frame.Content)
						fmt.Print(frame.Content)
					}
					for _, tc := range frame.ToolCalls {
						if tc.Function.Name != "" {
							fmt.Printf("\n  [call] %s(%s)\n%s", tc.Function.Name, tc.Function.Arguments, DJPrefix)
						}
					}
				}
			}
			continue
		}

		if mv.Message != nil {
			sb.WriteString(mv.Message.Content)
			fmt.Print(mv.Message.Content)
		}
	}
	return sb.String(), nil
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func drainMessage(mv *adk.MessageVariant) string {
	if mv.IsStreaming && mv.MessageStream != nil {
		var sb strings.Builder
		for {
			chunk, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				sb.WriteString(chunk.Content)
			}
		}
		return sb.String()
	}
	if mv.Message != nil {
		return mv.Message.Content
	}
	return ""
}
