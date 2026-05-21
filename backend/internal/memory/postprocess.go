package memory

import (
	"encoding/json"
	"strings"
)

// SongItem is a parsed song from agent JSON output.
type SongItem struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Segue  string `json:"segue,omitempty"`
}

// Postprocess parses agent output and records play history + updates preferences.
func (s *Store) Postprocess(sessionID, userPrompt, agentOutput string) {
	if sessionID == "" {
		return
	}

	trimmed := strings.TrimSpace(agentOutput)

	if strings.HasPrefix(trimmed, "[") {
		var items []SongItem
		if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
			return
		}
		for _, item := range items {
			_ = s.InsertPlay(&PlayRecord{
				SongTitle:  item.Title,
				SongArtist: item.Artist,
				UserPrompt: userPrompt,
				SessionID:  sessionID,
			})
			s.RecordPreferences(item.Artist, "", "")
		}

		var names []string
		for _, item := range items {
			names = append(names, item.Artist+"-"+item.Title)
		}
		_ = s.InsertTurn(sessionID, "assistant", "播放了："+strings.Join(names, "、"))
	} else {
		_ = s.InsertTurn(sessionID, "assistant", trimmed)
	}

	_ = s.InsertTurn(sessionID, "user", userPrompt)
	_ = s.TrimOldTurns(sessionID, maxConversationTurns)
}
