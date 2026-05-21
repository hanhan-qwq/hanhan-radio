package memory

import (
	"fmt"
	"strings"
)

// Preprocess loads all memory data and assembles SessionValues for ADK injection.
func (s *Store) Preprocess(sessionID string) map[string]any {
	return map[string]any{
		"UserPreferences":     s.GetPreferences(),
		"RecentPlays":         s.formatRecentPlays(),
		"ConversationContext": s.formatConversationContext(sessionID),
	}
}

func (s *Store) formatRecentPlays() string {
	records, err := s.RecentPlays()
	if err != nil || len(records) == 0 {
		return "暂无播放记录"
	}
	var lines []string
	for i, r := range records {
		line := fmt.Sprintf("%d. %s - %s", i+1, r.SongTitle, r.SongArtist)
		if r.SongGenre != "" {
			line += fmt.Sprintf("（%s）", r.SongGenre)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (s *Store) formatConversationContext(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	turns, err := s.RecentTurns(sessionID, 10)
	if err != nil || len(turns) == 0 {
		return ""
	}
	var lines []string
	for _, t := range turns {
		role := "用户"
		if t.Role == "assistant" {
			role = "助手"
		}
		content := t.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		lines = append(lines, fmt.Sprintf("%s：%s", role, content))
	}
	return "## 对话历史\n" + strings.Join(lines, "\n")
}
