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
func (h *Host) Generate(ctx context.Context, track playlist.Track, sel *selector.Result, last *LastPlayed, state, timeInfo, festival string) (*schema.StreamReader[*schema.Message], error) {
	var prompt string
	if last == nil {
		prompt = h.firstTpl
		prompt = strings.ReplaceAll(prompt, "{{song}}", track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", state)
		prompt = injectTime(prompt, timeInfo, festival)
	} else {
		prompt = h.nextTpl
		prompt = strings.ReplaceAll(prompt, "{{last_brief}}", last.Brief)
		prompt = strings.ReplaceAll(prompt, "{{last_song}}", last.Track.Song)
		prompt = strings.ReplaceAll(prompt, "{{last_artist}}", last.Track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{song}}", track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", state)
		prompt = injectTime(prompt, timeInfo, festival)
	}

	return h.cm.Stream(ctx, []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage("开始吧"),
	}, model.WithMaxTokens(1024))
}

func injectTime(prompt, timeInfo, festival string) string {
	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\n当前时间：")
	sb.WriteString(timeInfo)
	if festival != "" {
		sb.WriteString("，今天是")
		sb.WriteString(festival)
	}
	sb.WriteString("。")
	return sb.String()
}

type LastPlayed struct {
	Track playlist.Track
	Brief string
}

func Summary(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "..."
}
