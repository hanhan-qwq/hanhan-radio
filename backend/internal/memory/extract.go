package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const extractPrompt = `You are a music preference extractor. Analyze the conversation and extract structured facts about the user's music preferences.

## Existing Facts (do NOT repeat these)
%s

## Current Conversation
User said: %s
DJ played: %s

## Extraction Rules
1. Extract facts about: artists the user likes, genres they prefer, moods/scenarios that influence music choice (e.g. "tired → relaxing music"), listening context (time of day, activities), language preferences.
2. Assign confidence:
   - 0.7-0.9: user explicitly stated preference, or confirmed pattern across conversations
   - 0.4-0.6: reasonable inference from mood/context
   - 0.2-0.3: weak first-time signal, could be temporary
3. If the user shows dissatisfaction (skip, change, "not this"), extract a NEGATIVE fact with low confidence.
4. Each fact must be self-contained, under 30 words, in Chinese.
5. Do NOT repeat facts already covered by existing facts.
6. Return empty facts array if nothing new to extract. Only output valid JSON, no other text.

## Output Format
{"facts": [{"category": "artist_opinion|music_taste|mood_pattern|listening_habit", "content": "...", "confidence": 0.X}]}`

type extractResult struct {
	Facts []struct {
		Category   string  `json:"category"`
		Content    string  `json:"content"`
		Confidence float64 `json:"confidence"`
	} `json:"facts"`
}

// ExtractFacts calls an LLM to extract structured music preference facts
// from the user prompt and agent response, then writes new facts to the store.
func (s *Store) ExtractFacts(ctx context.Context, cm model.BaseChatModel, userPrompt, agentOutput string) error {
	if cm == nil {
		return nil
	}

	// Gather existing facts as context for dedup.
	existing, err := s.GetActiveFacts()
	if err != nil {
		return fmt.Errorf("get existing facts: %w", err)
	}
	existingStr := "none"
	if len(existing) > 0 {
		var lines []string
		for _, f := range existing {
			lines = append(lines, fmt.Sprintf("- [%s] %s (%.0f%%)", f.Category, f.Content, f.Confidence*100))
		}
		existingStr = strings.Join(lines, "\n")
	}

	// Build song summary from agent output.
	songInfo := agentOutput
	if strings.HasPrefix(strings.TrimSpace(agentOutput), "[") {
		var items []SongItem
		if err := json.Unmarshal([]byte(strings.TrimSpace(agentOutput)), &items); err == nil {
			var names []string
			for _, item := range items {
				names = append(names, fmt.Sprintf("%s - %s", item.Artist, item.Title))
			}
			songInfo = strings.Join(names, ", ")
		}
	}

	userPromptSafe := truncate(userPrompt, 200)
	userMsg := fmt.Sprintf(extractPrompt, existingStr, userPromptSafe, songInfo)

	messages := []*schema.Message{
		schema.SystemMessage("You are a JSON-only music preference extractor. Always output valid JSON."),
		schema.UserMessage(userMsg),
	}

	resp, err := cm.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("llm generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	// Strip markdown code fences if present.
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result extractResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return fmt.Errorf("parse extract result: %w (raw: %s)", err, truncate(content, 200))
	}

	for _, f := range result.Facts {
		if f.Content == "" {
			continue
		}
		cat := f.Category
		if cat != catArtistOpinion && cat != catMusicTaste && cat != catMoodPattern && cat != catListeningHabit {
			cat = catMusicTaste
		}
		if f.Confidence <= 0 {
			f.Confidence = 0.3
		}
		_ = s.AddFact(cat, f.Content, f.Confidence, "")
	}

	return nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
