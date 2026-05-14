package playlist

import (
	"fmt"
	"os"
	"strings"
)

type Song struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
}

func Load(path string) ([]Song, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read playlist: %w", err)
	}

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
		})
	}
	return songs, nil
}
