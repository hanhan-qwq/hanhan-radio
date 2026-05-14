package agent

import (
	"context"
	"sync"
	"time"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

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

// RadioHost runs the continuous radio loop using the graph pipeline.
type RadioHost struct {
	graph compose.Runnable[string, string]
	sess  *session.Session

	control chan string
	mu      sync.Mutex
	paused  bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewRadioHost(graph compose.Runnable[string, string], sess *session.Session) *RadioHost {
	ctx, cancel := context.WithCancel(context.Background())
	return &RadioHost{
		graph:   graph,
		sess:    sess,
		control: make(chan string, 8),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (h *RadioHost) Send(text string)  { h.control <- "speak:" + text }
func (h *RadioHost) Pause()             { h.control <- "pause" }
func (h *RadioHost) Resume()            { h.control <- "resume" }
func (h *RadioHost) Close()             { h.cancel() }

func (h *RadioHost) Start() <-chan Event {
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

func (h *RadioHost) runSegment(out chan<- Event) {
	msg := lastUserMessage(h.sess)

	result, err := h.graph.Invoke(h.ctx, msg)
	if err != nil {
		h.emit(out, "error", err.Error())
		return
	}

	// stream by sentence / chunk
	for _, chunk := range splitChunks(result, 30) {
		h.emit(out, "text", chunk)
		time.Sleep(30 * time.Millisecond)
	}
	h.emit(out, "done", "")
	h.sess.Append(schema.AssistantMessage(result, nil))
}

func (h *RadioHost) waitNext() string {
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

func (h *RadioHost) waitResume() string {
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

func (h *RadioHost) isPaused() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.paused
}
func (h *RadioHost) setPaused(v bool) { h.mu.Lock(); h.paused = v; h.mu.Unlock() }
func (h *RadioHost) stateLabel() string {
	if h.isPaused() { return "paused" }
	return "playing"
}
func (h *RadioHost) emit(out chan<- Event, typ, data string) {
	select {
	case out <- Event{Type: typ, Data: data}:
	default:
	}
}

// ── helpers ──

func lastUserMessage(sess *session.Session) string {
	msgs := sess.GetMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == schema.User {
			return msgs[i].Content
		}
	}
	return ""
}

func splitChunks(s string, n int) []string {
	runes := []rune(s)
	var chunks []string
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) { end = len(runes) }
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}


