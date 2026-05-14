package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	filePath  string
	mu        sync.Mutex
	messages  []*schema.Message
}

func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	data, _ := json.Marshal(msg)
	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", data)
	return nil
}

func (s *Session) GetMessages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

func (s *Session) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, msg := range s.messages {
		if msg.Role == schema.User && msg.Content != "" {
			t := msg.Content
			if len([]rune(t)) > 60 {
				t = string([]rune(t)[:60]) + "..."
			}
			return t
		}
	}
	return "New Session"
}

type Store struct {
	dir   string
	mu    sync.Mutex
	cache map[string]*Session
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	return &Store{dir: dir, cache: make(map[string]*Session)}, nil
}

func (s *Store) GetOrCreate(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}
	fp := filepath.Join(s.dir, id+".jsonl")
	var (
		sess *Session
		err  error
	)
	if _, statErr := os.Stat(fp); os.IsNotExist(statErr) {
		sess, err = createSession(id, fp)
	} else {
		sess, err = loadSession(fp)
	}
	if err != nil {
		return nil, err
	}
	s.cache[id] = sess
	return sess, nil
}

type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func createSession(id, fp string) (*Session, error) {
	h := sessionHeader{Type: "session", ID: id, CreatedAt: time.Now().UTC()}
	data, _ := json.Marshal(h)
	if err := os.WriteFile(fp, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	return &Session{ID: id, CreatedAt: h.CreatedAt, filePath: fp, messages: make([]*schema.Message, 0)}, nil
}

func loadSession(fp string) (*Session, error) {
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty file: %s", fp)
	}
	var h sessionHeader
	json.Unmarshal(scanner.Bytes(), &h)
	sess := &Session{ID: h.ID, CreatedAt: h.CreatedAt, filePath: fp, messages: make([]*schema.Message, 0)}
	for scanner.Scan() {
		var msg schema.Message
		if json.Unmarshal(scanner.Bytes(), &msg) == nil {
			sess.messages = append(sess.messages, &msg)
		}
	}
	return sess, scanner.Err()
}
