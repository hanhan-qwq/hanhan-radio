package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Stream wraps a runner.Run() call and emits output as events via a channel.
// Pure mechanism — no business logic.
type Stream struct {
	events chan Event
}

func NewStream() *Stream {
	return &Stream{events: make(chan Event, 64)}
}

func (s *Stream) Events() <-chan Event { return s.events }

// Run executes the agent and streams output. Closes the event channel when done.
func (s *Stream) Run(ctx context.Context, runner *adk.Runner, messages []*schema.Message) {
	defer close(s.events)

	events := runner.Run(ctx, messages)
	var content string

	for {
		event, ok := events.Next()
		if !ok {
			if content != "" {
				s.emit("done", "")
			}
			return
		}
		if event.Err != nil {
			s.emit("error", event.Err.Error())
			return
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
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
				if frame != nil && frame.Content != "" {
					content += frame.Content
					s.emit("text", frame.Content)
				}
			}
			continue
		}

		if mv.Message != nil {
			content += mv.Message.Content
			s.emit("text", mv.Message.Content)
		}
	}
}

func (s *Stream) emit(typ, data string) {
	select {
	case s.events <- Event{Type: typ, Data: data}:
	default:
	}
}
