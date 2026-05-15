package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
)

type Entry struct {
	Time  time.Time
	Track playlist.Track
	Brief string // 前 120 字串场词
}

type SessionMemory struct {
	Summary   string
	Recent    []Entry
	StartedAt time.Time
	SongCount int
	mu        sync.RWMutex
}

func New() *SessionMemory {
	return &SessionMemory{
		StartedAt: time.Now(),
		Recent:    make([]Entry, 0, 3),
	}
}

// Update async-calls LLM to update the rolling summary.
func (m *SessionMemory) Update(ctx context.Context, cm model.ToolCallingChatModel, entry Entry) {
	m.mu.Lock()
	m.SongCount++
	m.Recent = append([]Entry{entry}, m.Recent...)
	if len(m.Recent) > 3 {
		m.Recent = m.Recent[:3]
	}
	m.mu.Unlock()

	oldSummary := m.Summary

	go func() {
		prompt := fmt.Sprintf(updatePrompt, oldSummary, entry.Time.Format("15:04"),
			entry.Track.Song, entry.Track.Artist, entry.Track.Mood, entry.Brief)

		resp, err := cm.Generate(ctx, []*schema.Message{
			schema.SystemMessage("你是电台记忆管理助手。只输出摘要文本，不要额外解释。"),
			schema.UserMessage(prompt),
		})
		if err != nil {
			return
		}

		m.mu.Lock()
		m.Summary = resp.Content
		m.mu.Unlock()
	}()
}

// HostContext returns the full memory block for the DJ.
func (m *SessionMemory) HostContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SongCount == 0 {
		return ""
	}

	duration := time.Since(m.StartedAt).Round(time.Minute)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 今晚电台记忆\n已播歌曲数：%d\n", m.SongCount))
	sb.WriteString(fmt.Sprintf("电台开始于：%s，已持续约 %s\n\n", m.StartedAt.Format("15:04"), duration))

	if m.Summary != "" {
		sb.WriteString(m.Summary)
		sb.WriteString("\n\n")
	}

	sb.WriteString("最近播放：\n")
	for _, e := range m.Recent {
		sb.WriteString(fmt.Sprintf("- %s %s - %s (%s)\n",
			e.Time.Format("15:04"), e.Track.Song, e.Track.Artist, e.Track.Mood))
	}

	return sb.String()
}

// SelectorContext returns the simplified memory block for the selector.
func (m *SessionMemory) SelectorContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SongCount == 0 {
		return ""
	}

	var songs []string
	for _, e := range m.Recent {
		songs = append(songs, fmt.Sprintf("%s(%s)", e.Track.Song, e.Track.Mood))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("今晚已播(%d首): %s\n", m.SongCount, strings.Join(songs, "、")))

	if m.Summary != "" {
		sb.WriteString("\n情绪走势: ")
		// extract first line of summary as mood direction
		firstLine := strings.SplitN(m.Summary, "\n", 2)[0]
		sb.WriteString(firstLine)
	}

	return sb.String()
}

const updatePrompt = `你是电台记忆管理助手。根据当前摘要和新播放的歌曲，更新今晚电台摘要。

规则：
- 情绪走势：用一句话描述今晚的情绪变化方向
- 最近播放：列出最近播放的歌曲，包含歌名、歌手、情绪
- 如今天有节日/特殊事件，简要记录

当前摘要：
%s

新播放：
- 时间：%s
- 歌曲：%s - %s
- 情绪：%s
- 串场词要点：%s

输出更新后的完整摘要。只输出摘要文本，不要额外解释。`
