package radio

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/playlist"
	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

const autoContinueInterval = 4 * time.Second

var continuePrompts = []string{
	"（继续聊，推一首歌单里的歌）",
	"（继续说，自然的过渡到下一首歌）",
	"（接下来再推一首，不用打招呼了直接聊歌）",
	"（顺着刚才的氛围，再来一首）",
	"（说说你一直想推但还没推的那首歌）",
	"（聊一首你觉得被低估了的歌）",
}

var continueIdx int

func nextPrompt() string {
	s := continuePrompts[continueIdx%len(continuePrompts)]
	continueIdx++
	return s
}

type Event struct {
	Type string
	Data string
}

type Host struct {
	runner *adk.Runner
	sess   *session.Session

	control chan string
	mu      sync.Mutex
	paused  bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewHost(runner *adk.Runner, sess *session.Session) *Host {
	ctx, cancel := context.WithCancel(context.Background())
	return &Host{
		runner:  runner,
		sess:    sess,
		control: make(chan string, 8),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (h *Host) Send(text string)  { h.control <- "speak:" + text }
func (h *Host) Pause()             { h.control <- "pause" }
func (h *Host) Resume()            { h.control <- "resume" }
func (h *Host) Close()             { h.cancel() }

func (h *Host) Start() <-chan Event {
	out := make(chan Event, 64)
	h.sess.Append(schema.UserMessage("（电台开播了，打个招呼，然后推荐一首歌开始聊）"))

	go func() {
		defer close(out)
		h.emit(out, "state", "playing")

		for {
			h.emit(out, "state", h.stateLabel())
			h.runSegment(out)

			if h.isPaused() {
				h.emit(out, "state", "paused")
				next := h.waitResume()
				h.emit(out, "state", "playing")
				h.sess.Append(schema.UserMessage(next))
				continue
			}

			next := h.waitNext()
			if next == "" {
				return
			}
			h.sess.Append(schema.UserMessage(next))
		}
	}()

	return out
}

func (h *Host) runSegment(out chan<- Event) {
	events := h.runner.Run(h.ctx, h.sess.GetMessages())
	var content string

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			h.emit(out, "error", event.Err.Error())
			return
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// show tool results for debug
		if mv.Role == schema.Tool {
			if mv.Message != nil {
				h.emit(out, "tool", truncate(mv.Message.Content, 200))
			}
			continue
		}

		if mv.Role != schema.Assistant && mv.Role != "" {
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
					h.emit(out, "text", frame.Content)
				}
			}
			continue
		}

		if mv.Message != nil {
			content += mv.Message.Content
			h.emit(out, "text", mv.Message.Content)
		}
	}

	h.emit(out, "done", "")
	h.sess.Append(schema.AssistantMessage(content, nil))
}

func (h *Host) waitNext() string {
	timer := time.NewTimer(autoContinueInterval)
	defer timer.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return ""
		case c := <-h.control:
			if c == "pause" {
				h.setPaused(true)
				return ""
			}
			if len(c) > 6 && c[:6] == "speak:" {
				return c[6:]
			}
		case <-timer.C:
			return nextPrompt()
		}
	}
}

func (h *Host) waitResume() string {
	for {
		select {
		case <-h.ctx.Done():
			return ""
		case c := <-h.control:
			if c == "resume" || (len(c) > 6 && c[:6] == "speak:") {
				h.setPaused(false)
				if len(c) > 6 && c[:6] == "speak:" {
					return c[6:]
				}
				return "（听众回来了，自然的接上继续聊）"
			}
		}
	}
}

func (h *Host) isPaused() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.paused
}
func (h *Host) setPaused(v bool) { h.mu.Lock(); h.paused = v; h.mu.Unlock() }
func (h *Host) stateLabel() string {
	if h.isPaused() { return "paused" }
	return "playing"
}
func (h *Host) emit(out chan<- Event, typ, data string) {
	select {
	case out <- Event{Type: typ, Data: data}:
	default:
	}
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n] + "..."
}

type Config struct {
	ChatModel model.ToolCallingChatModel
	Songs     []playlist.Song
}

func BuildRunner(ctx context.Context, cfg Config) (*adk.Runner, error) {
	return buildDeepAgent(ctx, cfg)
}


