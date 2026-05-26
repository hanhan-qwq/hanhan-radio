package selectsong

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const rerankSystemPrompt = `你是一个音乐推荐助手。根据用户偏好从候选歌曲中选出最合适的一首。

必须严格按以下 JSON 格式输出：
{"title":"歌名","artist":"歌手","album":"","audio_url":"","duration":0,"genre":""}

选歌规则：
- 优先匹配用户的情绪、风格、语言、歌手偏好
- 参考用户画像中的长期偏好进行推荐
- 避开近期播放列表中的歌曲，除非用户明确要求重播
- 如果用户没有明确偏好，选择经典且广受欢迎的歌
- 从提供的候选歌曲中选取，不要编造不存在的歌`

// NewChain builds the semantic search + rerank chain:
//
//	build_query → hybrid_search → prompt_render → llm_rerank
func NewChain(ctx context.Context, cm model.BaseChatModel, searcher *Searcher, rctx *RerankContext) (compose.Runnable[*SelectSongInput, *SelectSongOutput], error) {
	c := compose.NewChain[*SelectSongInput, *SelectSongOutput]()

	// Step 1: build query text from structured input, embed, search.
	c.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *SelectSongInput) (*rerankInput, error) {
			queryText := buildQueryTextFromInput(in)
			candidates, err := searcher.Search(ctx, queryText)
			if err != nil {
				return nil, fmt.Errorf("hybrid search: %w", err)
			}
			if len(candidates) < minCandidatesRerank {
				return nil, fmt.Errorf("not enough candidates: got %d, need >= %d", len(candidates), minCandidatesRerank)
			}

			messages := buildRerankMessages(in, candidates, rctx)
			return &rerankInput{
				Input:      in,
				Candidates: candidates,
				Messages:   messages,
			}, nil
		}),
	)

	// Step 2: LLM rerank — pick the best song from candidates.
	c.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *rerankInput) (*SelectSongOutput, error) {
			msg, err := cm.Generate(ctx, in.Messages)
			if err != nil {
				return nil, fmt.Errorf("llm rerank: %w", err)
			}
			var out SelectSongOutput
			if err := json.Unmarshal([]byte(msg.Content), &out); err != nil {
				return nil, fmt.Errorf("parse rerank response: %w\ncontent: %s", err, msg.Content)
			}
			return &out, nil
		}),
	)

	return c.Compile(ctx)
}

func buildQueryTextFromInput(in *SelectSongInput) string {
	parts := []string{}
	if in.Artist != "" {
		parts = append(parts, in.Artist)
	}
	if in.Genre != "" {
		parts = append(parts, in.Genre)
	}
	if in.Mood != "" {
		parts = append(parts, in.Mood)
	}
	if in.Language != "" {
		parts = append(parts, in.Language)
	}
	if len(parts) == 0 {
		return "经典好歌"
	}
	return strings.Join(parts, " ")
}

// buildRerankMessages renders the full rerank prompt: user profile → recent plays → candidates → current preferences.
func buildRerankMessages(input *SelectSongInput, candidates []SearchCandidate, rctx *RerankContext) []*schema.Message {
	var sb strings.Builder

	// ── User Profile ──
	if rctx != nil && rctx.UserProfile != "" {
		sb.WriteString("## 用户画像（长期偏好）\n")
		sb.WriteString(rctx.UserProfile)
		sb.WriteString("\n\n")
	}

	// ── Recent Plays ──
	if rctx != nil && len(rctx.RecentPlays) > 0 {
		sb.WriteString("## 近期播放（请尽量避开以下歌曲，除非用户明确要求重播）\n")
		for _, rp := range rctx.RecentPlays {
			sb.WriteString(fmt.Sprintf("- %s\n", rp))
		}
		sb.WriteString("\n")
	}

	// ── Candidates with rich metadata ──
	sb.WriteString("## 候选歌曲\n")
	for i, c := range candidates {
		s := c.Song
		sb.WriteString(fmt.Sprintf("%d. %s - %s", i+1, s.Title, s.Artist))
		meta := []string{}
		if s.Genre != "" {
			meta = append(meta, fmt.Sprintf("风格: %s", s.Genre))
		}
		if s.Language != "" {
			meta = append(meta, fmt.Sprintf("语言: %s", s.Language))
		}
		if s.Year > 0 {
			meta = append(meta, fmt.Sprintf("年份: %d", s.Year))
		}
		meta = append(meta, fmt.Sprintf("相似度: %.0f%%", c.Similarity*100))
		if len(meta) > 0 {
			sb.WriteString(" | ")
			sb.WriteString(strings.Join(meta, " | "))
		}
		sb.WriteString("\n")
	}

	// ── Current preferences ──
	sb.WriteString("\n## 用户本次偏好\n")
	hasPref := false
	if input.Mood != "" {
		sb.WriteString(fmt.Sprintf("- 心情: %s\n", input.Mood))
		hasPref = true
	}
	if input.Genre != "" {
		sb.WriteString(fmt.Sprintf("- 风格: %s\n", input.Genre))
		hasPref = true
	}
	if input.Artist != "" {
		sb.WriteString(fmt.Sprintf("- 歌手: %s\n", input.Artist))
		hasPref = true
	}
	if input.Language != "" {
		sb.WriteString(fmt.Sprintf("- 语言: %s\n", input.Language))
		hasPref = true
	}
	if !hasPref {
		sb.WriteString("（无特殊偏好，请自由推荐一首经典好歌）\n")
	}

	return []*schema.Message{
		schema.SystemMessage(rerankSystemPrompt),
		schema.UserMessage(sb.String()),
	}
}
