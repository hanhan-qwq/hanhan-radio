package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/synthesizeaudio"
	"hanhan-radio/backend/internal/memory"
)

// EpisodeStatus is the lifecycle state of an episode.
type EpisodeStatus string

const (
	StatusPending    EpisodeStatus = "pending"
	StatusProcessing EpisodeStatus = "processing"
	StatusDone       EpisodeStatus = "done"
	StatusFailed     EpisodeStatus = "failed"
)

// Episode is a generated radio episode or chat response.
type Episode struct {
	ID        string            `json:"id"`
	Prompt    string            `json:"prompt"`
	Status    EpisodeStatus     `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	AudioURL  string            `json:"audio_url,omitempty"`
	Duration  int               `json:"duration,omitempty"`
	Segments  []SongSegmentItem `json:"segments,omitempty"`
	Message   string            `json:"message,omitempty"` // chat text when not a music request
	Error     string            `json:"error,omitempty"`
}

// SongSegmentItem is the public view of a SongSegment (without file_path).
type SongSegmentItem struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Segue  string `json:"segue,omitempty"`
}

const (
	baseDir   = "output"
	listLimit = 50
)

// Store holds episodes in memory.
type Store struct {
	mu   sync.RWMutex
	data map[string]*Episode
}

// NewStore creates an in-memory episode store.
func NewStore() *Store {
	return &Store{data: make(map[string]*Episode)}
}

// Create inserts a new episode.
func (s *Store) Create(ep *Episode) {
	s.mu.Lock()
	s.data[ep.ID] = ep
	s.mu.Unlock()
}

// Get returns an episode by id.
func (s *Store) Get(id string) (*Episode, bool) {
	s.mu.RLock()
	ep, ok := s.data[id]
	s.mu.RUnlock()
	return ep, ok
}

// List returns recent episodes (excluding failed), newest first.
func (s *Store) List() []*Episode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Episode
	for _, ep := range s.data {
		if ep.Status == StatusFailed {
			continue
		}
		result = append(result, ep)
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) > listLimit {
		result = result[:listLimit]
	}
	return result
}

// Update atomically updates an episode.
func (s *Store) Update(id string, fn func(*Episode)) {
	s.mu.Lock()
	if ep, ok := s.data[id]; ok {
		fn(ep)
	}
	s.mu.Unlock()
}

// Manager orchestrates episode creation and async agent execution.
type Manager struct {
	runner      *adk.Runner
	store       *Store
	memoryStore *memory.Store
	model       model.BaseChatModel // for LLM fact extraction
}

// New creates a Manager with the given runner, store, memory store, and optional model.
// model can be nil — fact extraction will be skipped.
func New(r *adk.Runner, s *Store, ms *memory.Store, cm model.BaseChatModel) *Manager {
	return &Manager{runner: r, store: s, memoryStore: ms, model: cm}
}

// Submit creates an episode and starts async execution.
func (m *Manager) Submit(ctx context.Context, prompt, sessionID string) (*Episode, error) {
	id, err := genID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	ep := &Episode{
		ID:        id,
		Prompt:    prompt,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}
	m.store.Create(ep)

	go m.execute(id, prompt, sessionID)

	return ep, nil
}

func (m *Manager) execute(id, prompt, sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	outDir := filepath.Join(baseDir, id)
	ctx = synthesizeaudio.WithOutputDir(ctx, outDir)

	// Pre-process: load memory and inject into agent instruction.
	if m.memoryStore != nil {
		values := m.memoryStore.Preprocess(sessionID)
		adk.AddSessionValues(ctx, values)
	}

	m.store.Update(id, func(ep *Episode) {
		ep.Status = StatusProcessing
	})

	log.L().Infow("episode_start", "id", id, "prompt", prompt)

	iter := m.runner.Query(ctx, prompt)
	content, err := consumeAgentOutput(iter)
	if err != nil {
		log.L().Errorw("episode_failed", "id", id, "err", err)
		m.store.Update(id, func(ep *Episode) {
			ep.Status = StatusFailed
			ep.Error = err.Error()
		})
		return
	}

	log.L().Debugw("agent_response", "id", id, "content", content)

	// Post-process: record play history and update preferences.
	if m.memoryStore != nil {
		m.memoryStore.Postprocess(sessionID, prompt, content)
		// LLM-based fact extraction from the conversation.
		if m.model != nil {
			if err := m.memoryStore.ExtractFacts(ctx, m.model, prompt, content); err != nil {
				log.L().Warnw("extract_facts_failed", "id", id, "err", err)
			}
		}
	}

	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "[") {
		m.handleMusicResponse(ctx, id, outDir, content)
	} else {
		m.handleChatResponse(id, content)
	}
}

func (m *Manager) handleMusicResponse(ctx context.Context, id, outDir, content string) {
	var items []SongSegmentItem
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		log.L().Errorw("parse_agent_output", "id", id, "err", err, "content", content)
		m.store.Update(id, func(ep *Episode) {
			ep.Status = StatusFailed
			ep.Error = fmt.Sprintf("parse agent output: %v", err)
		})
		return
	}

	audioPath := filepath.Join(outDir, "final.mp3")
	dur, err := synthesizeaudio.ProbeDuration(ctx, audioPath)
	if err != nil {
		log.L().Errorw("probe_duration", "id", id, "err", err)
		m.store.Update(id, func(ep *Episode) {
			ep.Status = StatusFailed
			ep.Error = fmt.Sprintf("probe audio duration: %v", err)
		})
		return
	}

	log.L().Infow("episode_done", "id", id, "audio", audioPath, "duration", dur, "segments", len(items))

	m.store.Update(id, func(ep *Episode) {
		ep.Status = StatusDone
		ep.AudioURL = "/static/" + id + "/final.mp3"
		ep.Duration = dur
		ep.Segments = items
	})
}

func (m *Manager) handleChatResponse(id, content string) {
	log.L().Infow("chat_done", "id", id, "content_len", len(content))

	m.store.Update(id, func(ep *Episode) {
		ep.Status = StatusDone
		ep.Message = content
	})
}

// consumeAgentOutput drains the async iterator and returns the final message content.
func consumeAgentOutput(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var lastContent string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil && msg != nil && msg.Content != "" {
				lastContent = msg.Content
			}
		}
	}

	if lastContent == "" {
		return "", fmt.Errorf("agent returned empty response")
	}
	return lastContent, nil
}

func genID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ep_" + hex.EncodeToString(b), nil
}
