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

## Last Played Song
%s

## Current Conversation
User said: %s
DJ said: %s

## Extraction Rules
1. Categories: artist_opinion (likes a specific artist), music_taste (genre/style preference), mood_pattern (mood→music mapping), listening_habit (repeated behavior).
2. User explicitly named an artist/genre/song in this conversation → extract artist_opinion or music_taste with confidence 0.7+.
3. User described a mood/scenario, DJ chose a song → extract only mood_pattern (0.4-0.6), NOT artist_opinion about the DJ's choice.
4. User gave positive/negative feedback ("这首不错""换一首""不太喜欢这首"):
   - Compare with Last Played Song to identify the target artist/genre.
   - Positive → bump confidence of existing matching fact, or add new fact (0.5-0.6).
   - Negative → extract a negative fact with low confidence (0.2-0.3).
5. User said "再来一首" without naming anything → NO extraction, return empty array. This just reinforces what was already played — no new signal.
6. User expressed music preference in casual chat ("我好喜欢周杰伦") → extract as if they named it explicitly (0.7+).
7. Each fact self-contained, under 30 words, in Chinese.
8. Do NOT repeat facts already in existing facts.
9. Return empty array if nothing new. Only output valid JSON, no other text.

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

	// Get last played song for reference resolution ("这首不错" etc.).
	lastPlayedStr := "none"
	plays, err := s.RecentPlays()
	if err == nil && len(plays) > 0 {
		last := plays[0]
		lastPlayedStr = fmt.Sprintf("%s - %s", last.SongArtist, last.SongTitle)
	}

	// Build DJ response summary.
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
	userMsg := fmt.Sprintf(extractPrompt, existingStr, lastPlayedStr, userPromptSafe, songInfo)

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
		if !validateFact(cat, f.Content, userPrompt, songInfo) {
			continue
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

// validateFact rejects facts that the LLM hallucinated — e.g. artist_opinion about an
// artist the user never mentioned. Checks both user prompt and agent output for context.
// mood_pattern and listening_habit are always accepted since they describe inferred patterns.
func validateFact(category, content, userPrompt, agentContext string) bool {
	switch category {
	case catArtistOpinion:
		return contentOverlapsPrompt(content, userPrompt) || contentOverlapsPrompt(content, agentContext)
	case catMusicTaste:
		return contentOverlapsPrompt(content, userPrompt) || contentOverlapsPrompt(content, agentContext)
	default:
		return true
	}
}

// contentOverlapsPrompt checks whether any >=2-rune word from fact content appears in the text.
func contentOverlapsPrompt(factContent, text string) bool {
	words := extractWords(factContent)
	for w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}
