package selector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
)

type Result struct {
	Track  playlist.Track
	Reason string
}

const selectPrompt = `你是一个深夜电台的音乐编辑。从歌单中选下一首要播放的歌。

当前时间: %s
%s
%s
%s
歌单:
%s

选一首最适合此刻的歌。输出 JSON:
{"song":"歌名","artist":"歌手","reason":"一句话选择理由"}

只输出 JSON。`

func Next(ctx context.Context, cm model.ToolCallingChatModel, tracks []playlist.Track, lastPlayed *playlist.Track, listenerState, timeInfo, festival string) (*Result, error) {
	var stateLine string
	if listenerState != "" {
		stateLine = fmt.Sprintf("听众状态: %s", listenerState)
	}

	var lastLine string
	if lastPlayed != nil {
		lastLine = fmt.Sprintf("上一首: %s - %s", lastPlayed.Song, lastPlayed.Artist)
	} else {
		lastLine = "这是今晚第一首歌"
	}

	var festivalLine string
	if festival != "" {
		festivalLine = fmt.Sprintf("今天是%s。", festival)
	}

	var trackList []string
	for _, t := range tracks {
		trackList = append(trackList, fmt.Sprintf("%s - %s [%s/%s]", t.Song, t.Artist, t.Mood, t.Style))
	}

	prompt := fmt.Sprintf(selectPrompt, timeInfo, festivalLine, lastLine, stateLine, strings.Join(trackList, "\n"))

	resp, err := cm.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是音乐编辑。只输出 JSON。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	return parse(resp.Content, tracks)
}

func parse(content string, tracks []playlist.Track) (*Result, error) {
	if i := strings.Index(content, "{"); i >= 0 {
		content = content[i:]
	}
	if i := strings.LastIndex(content, "}"); i >= 0 {
		content = content[:i+1]
	}

	var raw struct {
		Song   string `json:"song"`
		Artist string `json:"artist"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("parse selector output: %w", err)
	}

	// match back to a track
	for _, t := range tracks {
		if t.Song == raw.Song && t.Artist == raw.Artist {
			return &Result{Track: t, Reason: raw.Reason}, nil
		}
	}

	// fallback: fuzzy match by song name
	for _, t := range tracks {
		if strings.Contains(t.Song, raw.Song) || strings.Contains(raw.Song, t.Song) {
			return &Result{Track: t, Reason: raw.Reason}, nil
		}
	}

	return &Result{Track: tracks[0], Reason: "默认选择"}, nil
}
