package selectsong

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const queryParsePrompt = `Extract structured search criteria from this music request:

User request: %s

Output ONLY valid JSON:
{"mood":"","genre":"","artist":"","language":"","keywords":[],"bpm_fast":false,"bpm_slow":false}`

func parseQuery(ctx context.Context, cm model.BaseChatModel, userPrompt string) (*StructuredQuery, error) {
	msg := fmt.Sprintf(queryParsePrompt, truncatePrompt(userPrompt, 300))

	resp, err := cm.Generate(ctx, []*schema.Message{
		schema.SystemMessage("You are a music search query parser. Output only JSON. No other text."),
		schema.UserMessage(msg),
	})
	if err != nil {
		return nil, fmt.Errorf("llm parse query: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var sq StructuredQuery
	if err := json.Unmarshal([]byte(content), &sq); err != nil {
		return nil, fmt.Errorf("parse structured query json: %w (raw: %s)", err, truncatePrompt(content, 200))
	}
	return &sq, nil
}

func truncatePrompt(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// buildQueryText converts a StructuredQuery into a single string suitable for embedding.
func buildQueryText(sq *StructuredQuery) string {
	parts := []string{}
	if sq.Artist != "" {
		parts = append(parts, sq.Artist)
	}
	if sq.Genre != "" {
		parts = append(parts, sq.Genre)
	}
	if sq.Mood != "" {
		parts = append(parts, sq.Mood)
	}
	parts = append(parts, sq.Keywords...)
	if len(parts) == 0 {
		return "经典好歌"
	}
	return strings.Join(parts, " ")
}
