package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/synthesizeaudio"
)

// EpisodeStatus is the lifecycle state of an episode.
type EpisodeStatus string

const (
	StatusPending    EpisodeStatus = "pending"
	StatusProcessing EpisodeStatus = "processing"
	StatusDone       EpisodeStatus = "done"
	StatusFailed     EpisodeStatus = "failed"
)

// Episode is a generated radio episode.
type Episode struct {
	ID        string            `json:"id"`
	Prompt    string            `json:"prompt"`
	Status    EpisodeStatus     `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	AudioURL  string            `json:"audio_url,omitempty"`
	Duration  int               `json:"duration,omitempty"`
	Segments  []SongSegmentItem `json:"segments,omitempty"`
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
	agent *react.Agent
	store *Store
}

// New creates a Manager with the given agent and store.
func New(a *react.Agent, s *Store) *Manager {
	return &Manager{agent: a, store: s}
}

// Submit creates an episode and starts async execution.
func (m *Manager) Submit(ctx context.Context, prompt string) (*Episode, error) {
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

	go m.execute(id, prompt)

	return ep, nil
}

func (m *Manager) execute(id, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	outDir := filepath.Join(baseDir, id)
	ctx = synthesizeaudio.WithOutputDir(ctx, outDir)

	m.store.Update(id, func(ep *Episode) {
		ep.Status = StatusProcessing
	})

	log.L().Infow("episode_start", "id", id, "prompt", prompt)

	msg, err := m.agent.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		log.L().Errorw("episode_failed", "id", id, "err", err)
		m.store.Update(id, func(ep *Episode) {
			ep.Status = StatusFailed
			ep.Error = err.Error()
		})
		return
	}

	log.L().Debugw("agent_response", "id", id, "content", msg.Content)

	var items []SongSegmentItem
	if err := json.Unmarshal([]byte(msg.Content), &items); err != nil {
		log.L().Errorw("parse_agent_output", "id", id, "err", err, "content", msg.Content)
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

func genID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ep_" + hex.EncodeToString(b), nil
}
