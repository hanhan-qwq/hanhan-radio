package playlist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type Track struct {
	Song   string `json:"song"`
	Artist string `json:"artist"`
	Mood   string `json:"mood"`
	Style  string `json:"style"`
}

type rawTrack struct {
	Song   string
	Artist string
}

// Parse reads a playlist txt file (one "song  artist" per line) and returns raw tracks.
func Parse(path string) ([]rawTrack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read playlist: %w", err)
	}

	var tracks []rawTrack
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// split on first two spaces
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		tracks = append(tracks, rawTrack{
			Song:   strings.TrimSpace(parts[0]),
			Artist: strings.TrimSpace(parts[1]),
		})
	}
	return tracks, nil
}

const tagPrompt = `你是一个音乐标签系统。给以下歌曲打标签，输出 JSON 数组。

每首歌输出:
{"song": "歌名", "artist": "歌手", "mood": "情绪", "style": "风格"}

情绪枚举: 治愈/伤感/慵懒/活力/平静
风格: 1-2个词，如 民谣/R&B/流行/摇滚/电子

输入歌单:
%s

只输出 JSON 数组，不输出任何解释文字。`

// TagAndStore batch-tags raw tracks via LLM and saves to JSON.
func TagAndStore(ctx context.Context, cm model.ToolCallingChatModel, raw []rawTrack, outputPath string) ([]Track, error) {
	const batchSize = 10
	var all []Track

	for i := 0; i < len(raw); i += batchSize {
		end := i + batchSize
		if end > len(raw) {
			end = len(raw)
		}
		batch := raw[i:end]

		var trackList []string
		for _, t := range batch {
			trackList = append(trackList, fmt.Sprintf("%s %s", t.Song, t.Artist))
		}
		prompt := fmt.Sprintf(tagPrompt, strings.Join(trackList, "\n"))

		result, err := tagBatch(ctx, cm, prompt)
		if err != nil {
			// retry once
			result, err = tagBatch(ctx, cm, prompt)
			if err != nil {
				// skip batch, leave mood/style empty
				for _, t := range batch {
					all = append(all, Track{Song: t.Song, Artist: t.Artist})
				}
				continue
			}
		}
		all = append(all, result...)
	}

	data, _ := json.MarshalIndent(all, "", "  ")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write tracks.json: %w", err)
	}
	return all, nil
}

func tagBatch(ctx context.Context, cm model.ToolCallingChatModel, prompt string) ([]Track, error) {
	resp, err := cm.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是音乐标签系统。只输出 JSON 数组。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return nil, err
	}

	content := resp.Content
	if i := strings.Index(content, "["); i >= 0 {
		content = content[i:]
	}
	if i := strings.LastIndex(content, "]"); i >= 0 {
		content = content[:i+1]
	}

	var tracks []Track
	if err := json.Unmarshal([]byte(content), &tracks); err != nil {
		return nil, fmt.Errorf("parse tags: %w", err)
	}
	return tracks, nil
}

// Load reads tracks from a JSON file.
func Load(path string) ([]Track, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tracks []Track
	if err := json.Unmarshal(data, &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}
