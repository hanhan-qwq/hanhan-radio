package agent

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

// Event is a message emitted by the radio agent.
type Event struct {
	Type string // "text", "done", "error", "state"
	Data string
}

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

// RadioSession manages a continuous radio conversation.
type RadioSession struct {
	runner  *adk.Runner
	sess    *session.Session
	output  chan Event
	control chan string // "speak:xxx", "pause", "resume"
	ctx     context.Context
	cancel  context.CancelFunc

	mu     sync.Mutex
	paused bool
}

func NewRadioSession(runner *adk.Runner, sess *session.Session) *RadioSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &RadioSession{
		runner:  runner,
		sess:    sess,
		output:  make(chan Event, 64),
		control: make(chan string, 8),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Output returns the read-only event channel.
func (rs *RadioSession) Output() <-chan Event {
	return rs.output
}

// Send sends a user message to the radio.
func (rs *RadioSession) Send(text string) {
	rs.control <- "speak:" + text
}

// Pause pauses the auto-continue loop.
func (rs *RadioSession) Pause() {
	rs.control <- "pause"
}

// Resume resumes from paused state.
func (rs *RadioSession) Resume() {
	rs.control <- "resume"
}

// Close stops the radio session.
func (rs *RadioSession) Close() {
	rs.cancel()
	close(rs.output)
}

// Start begins the radio loop.
func (rs *RadioSession) Start() {
	rs.emit("state", "playing")
	rs.sess.Append(schema.UserMessage("（电台开播了，打个招呼，然后推荐一首歌开始聊）"))
	go rs.loop()
}

func (rs *RadioSession) emit(typ, data string) {
	select {
	case rs.output <- Event{Type: typ, Data: data}:
	case <-rs.ctx.Done():
	}
}

func (rs *RadioSession) loop() {
	defer close(rs.output)

	for {
		rs.mu.Lock()
		p := rs.paused
		rs.mu.Unlock()
		rs.emit("state", stateStr(p))

		rs.runSegment()
		rs.emit("done", "")

		// wait
		if p {
			rs.emit("state", "paused")
			rs.waitResume()
			rs.emit("state", "playing")
			rs.sess.Append(schema.UserMessage("（听众说话了，自然的接上继续聊）"))
			continue
		}

		timer := time.NewTimer(4 * time.Second)
		select {
		case <-rs.ctx.Done():
			timer.Stop()
			return
		case c := <-rs.control:
			timer.Stop()
			if c == "pause" {
				rs.mu.Lock()
				rs.paused = true
				rs.mu.Unlock()
				continue
			}
			if len(c) > 6 && c[:6] == "speak:" {
				rs.sess.Append(schema.UserMessage(c[6:]))
				continue
			}
		case <-timer.C:
			rs.sess.Append(schema.UserMessage(nextContinuePrompt()))
			continue
		}
	}
}

func (rs *RadioSession) waitResume() {
	for {
		select {
		case <-rs.ctx.Done():
			return
		case c := <-rs.control:
			if c == "resume" {
				rs.mu.Lock()
				rs.paused = false
				rs.mu.Unlock()
				return
			}
			if len(c) > 6 && c[:6] == "speak:" {
				rs.mu.Lock()
				rs.paused = false
				rs.mu.Unlock()
				rs.sess.Append(schema.UserMessage(c[6:]))
				return
			}
		}
	}
}

func (rs *RadioSession) runSegment() {
	events := rs.runner.Run(rs.ctx, rs.sess.GetMessages())
	var content string

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			rs.emit("error", event.Err.Error())
			return
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role == schema.Tool {
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
					rs.emit("text", frame.Content)
					// check pause during streaming
					select {
					case c := <-rs.control:
						if c == "pause" {
							rs.mu.Lock()
							rs.paused = true
							rs.mu.Unlock()
						}
					default:
					}
				}
			}
			continue
		}

		if mv.Message != nil {
			content += mv.Message.Content
			rs.emit("text", mv.Message.Content)
		}
	}

	rs.sess.Append(schema.AssistantMessage(content, nil))
}

func stateStr(paused bool) string {
	if paused {
		return "paused"
	}
	return "playing"
}
