package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/memory"
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

type Input struct {
	Track     playlist.Track
	Sel       *selector.Result
	Last      *LastPlayed
	State     string
	TimeInfo  string
	Festival  string
	Memory    *memory.SessionMemory
}

func (h *Host) Generate(ctx context.Context, in Input) (*schema.StreamReader[*schema.Message], error) {
	var prompt string
	if in.Last == nil {
		prompt = h.firstTpl
		prompt = strings.ReplaceAll(prompt, "{{song}}", in.Track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", in.Track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", in.Sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", in.State)
		prompt = injectContext(prompt, in)
	} else {
		prompt = h.nextTpl
		prompt = strings.ReplaceAll(prompt, "{{last_brief}}", in.Last.Brief)
		prompt = strings.ReplaceAll(prompt, "{{last_song}}", in.Last.Track.Song)
		prompt = strings.ReplaceAll(prompt, "{{last_artist}}", in.Last.Track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{song}}", in.Track.Song)
		prompt = strings.ReplaceAll(prompt, "{{artist}}", in.Track.Artist)
		prompt = strings.ReplaceAll(prompt, "{{reason}}", in.Sel.Reason)
		prompt = strings.ReplaceAll(prompt, "{{state}}", in.State)
		prompt = injectContext(prompt, in)
	}

	return h.cm.Stream(ctx, []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage("开始吧"),
	}, model.WithMaxTokens(1024))
}

func injectContext(prompt string, in Input) string {
	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\n当前时间：")
	sb.WriteString(in.TimeInfo)
	if in.Festival != "" {
		sb.WriteString("，今天是")
		sb.WriteString(in.Festival)
	}
	sb.WriteString("。")

	if mem := in.Memory.HostContext(); mem != "" {
		sb.WriteString("\n\n")
		sb.WriteString(mem)
	}

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
