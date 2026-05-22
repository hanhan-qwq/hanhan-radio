package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const consolidatePrompt = `You are a knowledge graph curator. Merge the following music preference facts into a clean, deduplicated list.

## Rules
1. Combine facts that express the same idea into one — keep the most informative wording.
2. When merging, take the highest confidence value and note it.
3. Remove facts that are contradicted by higher-confidence facts.
4. Keep each fact self-contained, under 30 words, in Chinese.
5. Return ALL facts (both kept originals and merged ones). Output the full cleaned list.

## Facts to Consolidate
%s

## Output Format
{"facts": [{"category": "...", "content": "...", "confidence": 0.X}]}`

const consolidateThreshold = 15

// ConsolidateFacts calls the LLM to deduplicate and merge similar facts.
// Returns nil (and logs a warning) if model is nil or no consolidation is needed.
func (s *Store) ConsolidateFacts(ctx context.Context, cm model.BaseChatModel) error {
	if cm == nil {
		return nil
	}

	facts, err := s.GetActiveFacts()
	if err != nil {
		return fmt.Errorf("get active facts: %w", err)
	}
	if len(facts) <= consolidateThreshold {
		return nil
	}

	var lines []string
	for _, f := range facts {
		lines = append(lines, fmt.Sprintf("- [%s] %s (%.0f%%)", f.Category, f.Content, f.Confidence*100))
	}

	userMsg := fmt.Sprintf(consolidatePrompt, strings.Join(lines, "\n"))
	messages := []*schema.Message{
		schema.SystemMessage("You are a JSON-only fact consolidator. Always output valid JSON."),
		schema.UserMessage(userMsg),
	}

	resp, err := cm.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("llm generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result struct {
		Facts []struct {
			Category   string  `json:"category"`
			Content    string  `json:"content"`
			Confidence float64 `json:"confidence"`
		} `json:"facts"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return fmt.Errorf("parse consolidate result: %w (raw: %s)", err, truncate(content, 200))
	}

	// Build a map from LLM output content to confidence for fast lookup.
	kept := map[string]float64{}
	for _, f := range result.Facts {
		kept[strings.TrimSpace(f.Content)] = f.Confidence
	}

	// Expire old facts not in the consolidated list.
	for _, old := range facts {
		newConf, ok := kept[strings.TrimSpace(old.Content)]
		if ok {
			// Update confidence if the LLM suggests a higher one.
			if newConf > old.Confidence {
				old.Confidence = newConf
				_ = s.DB.Save(&old).Error
			}
		} else {
			_ = s.ExpireFact(old.ID)
		}
	}

	// Insert any new merged facts from the LLM output.
	for _, f := range result.Facts {
		content := strings.TrimSpace(f.Content)
		if content == "" {
			continue
		}
		// Check if this fact already exists (avoid re-insert).
		existing, _ := s.findSimilarFact(f.Category, content)
		if existing != nil {
			continue
		}
		cat := f.Category
		if cat != catArtistOpinion && cat != catMusicTaste && cat != catMoodPattern && cat != catListeningHabit {
			cat = catMusicTaste
		}
		_ = s.DB.Create(&MemoryFact{
			Category:   cat,
			Content:    content,
			Confidence: f.Confidence,
		}).Error
	}

	return nil
}
