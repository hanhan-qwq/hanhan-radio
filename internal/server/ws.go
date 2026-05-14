package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var continuePrompts = []string{
	"（继续你的节目，推一首歌聊聊）",
	"（上一段结束了，自然的接下去，再推一首歌单里的歌）",
	"（听众还在，继续你的深夜电台，聊聊下一首歌）",
	"（顺着刚才的氛围，再来一首，不用打招呼了直接聊）",
	"（聊一首歌单里的经典，说说你为什么选它）",
}

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// controlMsg represents a user action from the frontend
type controlMsg struct {
	kind string // "speak", "pause", "resume"
	text string // only for "speak"
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	sess, _ := s.store.GetOrCreate("ws-" + randomID())

	control := make(chan controlMsg, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// reader goroutine
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
			var m wsMessage
			if json.Unmarshal(msg, &m) != nil {
				continue
			}
			switch m.Type {
			case "speak":
				if m.Data != "" {
					control <- controlMsg{kind: "speak", text: m.Data}
				}
			case "pause":
				control <- controlMsg{kind: "pause"}
			case "resume":
				control <- controlMsg{kind: "resume"}
			}
		}
	}()

	// start the first segment
	_ = sess.Append(schema.UserMessage("（电台开播了，开始你的节目，打个招呼然后推一首歌）"))
	runLoop(ctx, s.runner, conn, sess, control)
}

var continueIdx int

func nextContinuePrompt() string {
	s := continuePrompts[continueIdx%len(continuePrompts)]
	continueIdx++
	return s
}

func runLoop(ctx context.Context, runner *adk.Runner, conn *websocket.Conn, sess *session.Session, control <-chan controlMsg) {
	var mu sync.Mutex
	paused := false

	for {
		writeWS(&mu, conn, "state", stateStr(paused))

		events := runner.Run(ctx, sess.GetMessages())
		var content strings.Builder

	loop:
		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				writeWS(&mu, conn, "error", event.Err.Error())
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
						break loop
					}
					if frame != nil && frame.Content != "" {
						content.WriteString(frame.Content)
						writeWS(&mu, conn, "text", frame.Content)
						// check for pause during streaming
						select {
						case c := <-control:
							if c.kind == "pause" {
								paused = true
								writeWS(&mu, conn, "state", "paused")
							} else if c.kind == "resume" {
								paused = false
								writeWS(&mu, conn, "state", "playing")
							}
						default:
						}
					}
				}
				continue
			}

			if mv.Message != nil {
				content.WriteString(mv.Message.Content)
				writeWS(&mu, conn, "text", mv.Message.Content)
			}
		}

		_ = sess.Append(schema.AssistantMessage(content.String(), nil))
		writeWS(&mu, conn, "done", "")

		// wait for next action
		if paused {
			writeWS(&mu, conn, "state", "paused")
			// wait for resume or speak
			for {
				select {
				case <-ctx.Done():
					return
				case c := <-control:
					if c.kind == "resume" {
						paused = false
						writeWS(&mu, conn, "state", "playing")
						goto next
					}
					if c.kind == "speak" {
						paused = false
						writeWS(&mu, conn, "state", "playing")
						_ = sess.Append(schema.UserMessage(c.text))
						goto next
					}
				}
			}
		next:
			_ = sess.Append(schema.UserMessage("（听众回来了，继续你的节目，自然的接上）"))
			continue
		}

		// not paused: auto-continue or wait for user
		timer := time.NewTimer(4 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case c := <-control:
			timer.Stop()
			if c.kind == "pause" {
				paused = true
				continue
			}
			if c.kind == "speak" {
				_ = sess.Append(schema.UserMessage(c.text))
				continue
			}
		case <-timer.C:
			_ = sess.Append(schema.UserMessage(nextContinuePrompt()))
			continue
		}
	}
}

func stateStr(paused bool) string {
	if paused {
		return "paused"
	}
	return "playing"
}

func writeWS(mu *sync.Mutex, conn *websocket.Conn, typ, data string) {
	mu.Lock()
	defer mu.Unlock()
	b, _ := json.Marshal(wsMessage{Type: typ, Data: data})
	conn.WriteMessage(websocket.TextMessage, b)
}

func randomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}
