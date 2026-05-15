package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/selector"
)

const defaultState = "深夜，听众想听一首歌放松一下"

type LastPlayed struct {
	Track playlist.Track
	Brief string
}

type Host struct {
	cm       model.ToolCallingChatModel
	firstTpl string
	nextTpl  string
}

func New(cm model.ToolCallingChatModel, promptsDir string) (*Host, error) {
	first, err := os.ReadFile(filepath.Join(promptsDir, "first.txt"))
	if err != nil {
		return nil, fmt.Errorf("read first.txt: %w", err)
	}
	next, err := os.ReadFile(filepath.Join(promptsDir, "next.txt"))
	if err != nil {
		return nil, fmt.Errorf("read next.txt: %w", err)
	}
	return &Host{cm: cm, firstTpl: string(first), nextTpl: string(next)}, nil
}

// Generate produces a streaming script for the given track.
func (h *Host) Generate(ctx context.Context, track playlist.Track, sel *selector.Result, last *LastPlayed, state string) (*schema.StreamReader[*schema.Message], error) {
	if state == "" {
		state = defaultState
	}

	var prompt string
	if last == nil {
		prompt = h.firstTpl
		prompt = strings.ReplaceAll(prompt, "{{song}}", track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", state)
	} else {
		prompt = h.nextTpl
		prompt = strings.ReplaceAll(prompt, "{{last_brief}}", last.Brief)
		prompt = strings.ReplaceAll(prompt, "{{last_song}}", last.Track.Song)
		prompt = strings.ReplaceAll(prompt, "{{last_artist}}", last.Track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{song}}", track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", state)
	}

	return h.cm.Stream(ctx, []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage("开始吧"),
	})
}

// Summary returns the first n runes of text.
func Summary(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "..."
}
