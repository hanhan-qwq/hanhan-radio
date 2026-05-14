package agent

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

const (
	autoContinueInterval = 4 * time.Second
)

var continuePrompts = []string{
	"（继续聊，推一首歌单里的歌）",
	"（继续说，自然的过渡到下一首歌）",
	"（接下来再推一首，不用打招呼了直接聊歌）",
	"（顺着刚才的氛围，再来一首）",
	"（说说你一直想推但还没推的那首歌）",
	"（聊一首你觉得被低估了的歌）",
}

var continueIdx int

func nextContinuePrompt() string {
	s := continuePrompts[continueIdx%len(continuePrompts)]
	continueIdx++
	return s
}

// RadioHost manages a continuous radio conversation loop.
type RadioHost struct {
	runner *adk.Runner
	sess   *session.Session

	control chan string // "speak:xxx", "pause", "resume"

	mu     sync.Mutex
	paused bool
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRadioHost(runner *adk.Runner, sess *session.Session) *RadioHost {
	ctx, cancel := context.WithCancel(context.Background())
	return &RadioHost{
		runner:  runner,
		sess:    sess,
		control: make(chan string, 8),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Send forwards a user message.
func (h *RadioHost) Send(text string) { h.control <- "speak:" + text }

// Pause pauses auto-continue after current segment finishes.
func (h *RadioHost) Pause() { h.control <- "pause" }

// Resume resumes from paused state.
func (h *RadioHost) Resume() { h.control <- "resume" }

// Close stops the radio loop.
func (h *RadioHost) Close() { h.cancel() }

// Start begins the radio loop. Returns a channel of output events.
func (h *RadioHost) Start() <-chan Event {
	out := make(chan Event, 64)

	h.sess.Append(schema.UserMessage("（电台开播了，打个招呼，然后推荐一首歌开始聊）"))

	go func() {
		defer close(out)

		h.emit(out, "state", "playing")

		for {
			h.emit(out, "state", h.stateLabel())

			// run one segment
			st := NewStream()
			go st.Run(h.ctx, h.runner, h.sess.GetMessages())

			var content string
			for evt := range st.Events() {
				if evt.Type == "text" {
					content += evt.Data
				}
				if evt.Type == "done" {
					h.sess.Append(schema.AssistantMessage(content, nil))
				}
				out <- evt
			}

			// decide what next
			if h.isPaused() {
				h.emit(out, "state", "paused")
				next := h.waitResume(out)
				h.emit(out, "state", "playing")
				h.sess.Append(schema.UserMessage(next))
				continue
			}

			next := h.waitNext(out)
			if next == "" {
				return
			}
			h.sess.Append(schema.UserMessage(next))
		}
	}()

	return out
}

func (h *RadioHost) waitNext(out chan<- Event) string {
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
			return nextContinuePrompt()
		}
	}
}

func (h *RadioHost) waitResume(out chan<- Event) string {
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

func (h *RadioHost) setPaused(v bool) {
	h.mu.Lock()
	h.paused = v
	h.mu.Unlock()
}

func (h *RadioHost) stateLabel() string {
	if h.isPaused() {
		return "paused"
	}
	return "playing"
}

func (h *RadioHost) emit(out chan<- Event, typ, data string) {
	select {
	case out <- Event{Type: typ, Data: data}:
	default:
	}
}
