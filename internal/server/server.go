package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/internal/session"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	runner  *adk.Runner
	store   *session.Store
}

func New(runner *adk.Runner, store *session.Store) *Server {
	return &Server{runner: runner, store: store}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// static files
	static, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))

	// api
	mux.HandleFunc("/api/chat", s.handleChat)

	log.Printf("电台服务启动: http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sess, err := s.store.GetOrCreate(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// save user message
	_ = sess.Append(schema.UserMessage(req.Message))

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// run agent
	events := s.runner.Run(r.Context(), sess.GetMessages())
	var fullContent strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			sendSSE(w, flusher, "error", event.Err.Error())
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
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					sendSSE(w, flusher, "error", err.Error())
					return
				}
				if frame != nil && frame.Content != "" {
					fullContent.WriteString(frame.Content)
					sendSSE(w, flusher, "text", frame.Content)
				}
			}
			continue
		}

		if mv.Message != nil {
			fullContent.WriteString(mv.Message.Content)
			sendSSE(w, flusher, "text", mv.Message.Content)
		}
	}

	// save assistant message
	_ = sess.Append(schema.AssistantMessage(fullContent.String(), nil))

	// done
	sendSSE(w, flusher, "done", sess.Title())
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	// escape newlines in data for SSE
	data = strings.ReplaceAll(data, "\n", " ")
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}

