package memory

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	catArtistOpinion  = "artist_opinion"
	catMusicTaste     = "music_taste"
	catMoodPattern    = "mood_pattern"
	catListeningHabit = "listening_habit"
)

// AddFact inserts a new fact or merges with a similar existing one.
// If the new fact has very low confidence (<0.3), it's treated as a negative signal
// and penalizes matching positive facts instead of merging.
func (s *Store) AddFact(category, content string, confidence float64, source string) error {
	// Negative signal: penalize matching positive facts.
	if confidence < 0.3 {
		return s.penalizeMatchingFacts(category, content)
	}

	similar, err := s.findSimilarFact(category, content)
	if err != nil {
		return err
	}
	if similar != nil {
		similar.Confidence = math.Min(similar.Confidence+confidence*0.3, 1.0)
		if source != "" {
			similar.Source = source
		}
		return s.DB.Save(similar).Error
	}

	return s.DB.Create(&MemoryFact{
		Category:   category,
		Content:    content,
		Confidence: confidence,
		Source:     source,
	}).Error
}

// penalizeMatchingFacts finds positive facts in the same category with keyword overlap
// and reduces their confidence. Facts that drop below 0.3 are expired.
func (s *Store) penalizeMatchingFacts(category, content string) error {
	negWords := extractWords(content)

	var candidates []MemoryFact
	if err := s.DB.Where("category = ? AND expired_at = 0 AND confidence > 0.3", category).
		Find(&candidates).Error; err != nil {
		return err
	}

	for i := range candidates {
		if wordOverlap(negWords, candidates[i].Content) > 0 {
			candidates[i].Confidence = max(candidates[i].Confidence-0.2, 0.1)
			if candidates[i].Confidence < 0.3 {
				candidates[i].ExpiredAt = time.Now().UnixMilli()
			}
			if err := s.DB.Save(&candidates[i]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// findSimilarFact returns an active fact in the same category with overlapping keywords.
func (s *Store) findSimilarFact(category, content string) (*MemoryFact, error) {
	var facts []MemoryFact
	if err := s.DB.Where("category = ? AND expired_at = 0", category).Find(&facts).Error; err != nil {
		return nil, err
	}

	words := extractWords(content)
	for i := range facts {
		if wordOverlap(words, facts[i].Content) > 0.5 {
			return &facts[i], nil
		}
	}
	return nil, nil
}

// delimiters that separate meaningful words in Chinese text.
var delimReplacer = strings.NewReplacer(
	"，", " ",
	"。", " ",
	"、", " ",
	"的", " ",
	"了", " ",
	"是", " ",
	"在", " ",
	"很", " ",
	"非常", " ",
)

func extractWords(s string) map[string]bool {
	cleaned := delimReplacer.Replace(s)
	words := make(map[string]bool)
	for _, w := range strings.Fields(cleaned) {
		if len([]rune(w)) >= 2 {
			words[w] = true
		}
	}
	return words
}

func wordOverlap(a map[string]bool, b string) float64 {
	if len(a) == 0 {
		return 0
	}
	bWords := extractWords(b)
	var overlap int
	for w := range a {
		if bWords[w] {
			overlap++
		}
	}
	return float64(overlap) / float64(len(a))
}

// ExpireFact marks a fact as expired.
func (s *Store) ExpireFact(id uint) error {
	return s.DB.Model(&MemoryFact{}).Where("id = ?", id).
		Update("expired_at", time.Now().UnixMilli()).Error
}

// GetActiveFacts returns all non-expired facts, optionally filtered by category.
func (s *Store) GetActiveFacts(categories ...string) ([]MemoryFact, error) {
	q := s.DB.Where("expired_at = 0")
	if len(categories) > 0 {
		q = q.Where("category IN ?", categories)
	}
	var facts []MemoryFact
	err := q.Order("confidence DESC").Find(&facts).Error
	return facts, err
}

// RecordArtistPlay records a fact that the user likes a specific artist.
func (s *Store) RecordArtistPlay(artist string) {
	if artist == "" {
		return
	}
	content := fmt.Sprintf("用户喜欢听%s的歌", artist)
	_ = s.AddFact(catArtistOpinion, content, 0.3, "")
}

// RecordGenrePlay records a fact that the user likes a specific genre.
func (s *Store) RecordGenrePlay(genre string) {
	if genre == "" {
		return
	}
	content := fmt.Sprintf("用户喜欢%s风格的音乐", genre)
	_ = s.AddFact(catMusicTaste, content, 0.3, "")
}

// RecordPreferences records artist and genre preferences from a play.
// Kept for backward compatibility — delegates to fact-based recording.
func (s *Store) RecordPreferences(artist, genre, language string) {
	s.RecordArtistPlay(artist)
	s.RecordGenrePlay(genre)
}

// decayFactor returns the time decay multiplier for a fact.
// Confidence decays ~0.5% per day, bottoming at 30% of original.
func (s *Store) decayFactor(f MemoryFact) float64 {
	days := float64(time.Now().UnixMilli()-f.UpdatedAt) / 86400000.0
	if days < 1 {
		return 1.0
	}
	return max(0.3, 1.0-days*0.005)
}

// GetProfileSummary generates the L1 user profile text for prompt injection.
func (s *Store) GetProfileSummary() string {
	facts, err := s.GetActiveFacts()
	if err != nil || len(facts) == 0 {
		return ""
	}

	groups := map[string][]MemoryFact{}
	for _, f := range facts {
		groups[f.Category] = append(groups[f.Category], f)
	}

	catNames := map[string]string{
		catArtistOpinion:  "歌手偏好",
		catMusicTaste:     "音乐口味",
		catMoodPattern:    "心情模式",
		catListeningHabit: "收听习惯",
	}

	var parts []string
	for _, cat := range []string{catArtistOpinion, catMusicTaste, catMoodPattern, catListeningHabit} {
		fs := groups[cat]
		if len(fs) == 0 {
			continue
		}
		sort.Slice(fs, func(i, j int) bool { return fs[i].Confidence > fs[j].Confidence })
		top := fs
		if len(top) > 5 {
			top = top[:5]
		}
		var items []string
		for _, f := range top {
			effective := f.Confidence * s.decayFactor(f)
			if effective >= 0.3 {
				items = append(items, f.Content)
			}
		}
		if len(items) > 0 {
			name := catNames[cat]
			if name == "" {
				name = cat
			}
			parts = append(parts, name+"："+strings.Join(items, "；"))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "【用户画像】" + strings.Join(parts, "。") + "。选歌时请优先参考这些信息。"
}
