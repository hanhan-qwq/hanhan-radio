package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// Event is a message emitted by the radio agent.
type Event struct {
	Type string // "text", "done", "error", "state"
	Data string
}

type Song struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Index  int    `json:"index"`
}

// LoadSongs reads and parses the playlist file.
func LoadSongs(path string) ([]Song, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read playlist: %w", err)
	}
	return parseSongs(data), nil
}

func parseSongs(data []byte) []Song {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var songs []Song
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ".")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+1:])
		parts := strings.SplitN(rest, "||", 2)
		if len(parts) != 2 {
			continue
		}
		songs = append(songs, Song{
			Name:   strings.TrimSpace(parts[0]),
			Artist: strings.TrimSpace(parts[1]),
			Index:  len(songs) + 1,
		})
	}
	return songs
}

// BuildGraph creates the radio pipeline graph.
func BuildGraph(ctx context.Context, chatModel model.ToolCallingChatModel) (compose.Runnable[string, string], error) {
	songs, err := LoadSongs("data/playlist.txt")
	if err != nil {
		return nil, err
	}
	return buildRadioGraph(ctx, chatModel, songs)
}

var _ = json.Marshal
