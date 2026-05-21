package memory

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const decayFactor = 0.95

// bumpScore increments a profile score by delta, capped to 1.0.
func (s *Store) bumpScore(key string, delta float64) error {
	var entry ProfileEntry
	err := s.DB.First(&entry, "key = ?", key).Error
	if err != nil {
		entry = ProfileEntry{Key: key, Value: 0}
	}
	entry.Value = math.Min(entry.Value+delta, 1.0)
	return s.DB.Save(&entry).Error
}

// RecordPreferences bumps artist and genre scores from a play.
func (s *Store) RecordPreferences(artist, genre, language string) {
	if artist != "" {
		_ = s.bumpScore("artist:"+artist, 0.1)
	}
	if genre != "" {
		_ = s.bumpScore("genre:"+genre, 0.1)
	}
	if language != "" {
		_ = s.bumpScore("lang:"+language, 0.1)
	}
}

// ApplyDecay multiplies all profile values by the decay factor.
// Should be called at most once per day.
func (s *Store) ApplyDecay() error {
	var entries []ProfileEntry
	if err := s.DB.Find(&entries).Error; err != nil {
		return err
	}
	for _, e := range entries {
		e.Value = math.Max(e.Value*decayFactor, 0.01)
		if err := s.DB.Save(&e).Error; err != nil {
			return err
		}
	}
	return nil
}

// LastDecay returns the timestamp of the last decay operation.
// Returns zero if never decayed.
func (s *Store) LastDecay() int64 {
	var entry ProfileEntry
	err := s.DB.First(&entry, "key = ?", "_decay_ts").Error
	if err != nil {
		return 0
	}
	return int64(entry.Value)
}

// SetLastDecay records the decay timestamp.
func (s *Store) SetLastDecay(ts int64) {
	entry := ProfileEntry{Key: "_decay_ts", Value: float64(ts)}
	_ = s.DB.Save(&entry)
}

// MaybeDecay applies decay if at least 24 hours have passed since the last decay.
func (s *Store) MaybeDecay() error {
	last := s.LastDecay()
	now := time.Now().UnixMilli()
	if now-last > 24*3600*1000 {
		if err := s.ApplyDecay(); err != nil {
			return err
		}
		s.SetLastDecay(now)
	}
	return nil
}

// GetPreferences returns the profile as a human-readable prompt string.
func (s *Store) GetPreferences() string {
	_ = s.MaybeDecay()

	var entries []ProfileEntry
	if err := s.DB.Where("key NOT LIKE ?", "_%").Find(&entries).Error; err != nil {
		return ""
	}
	if len(entries) == 0 {
		return ""
	}

	artists := map[string]float64{}
	genres := map[string]float64{}
	langs := map[string]float64{}

	for _, e := range entries {
		switch {
		case strings.HasPrefix(e.Key, "artist:"):
			artists[e.Key[7:]] = e.Value
		case strings.HasPrefix(e.Key, "genre:"):
			genres[e.Key[6:]] = e.Value
		case strings.HasPrefix(e.Key, "lang:"):
			langs[e.Key[5:]] = e.Value
		}
	}

	var parts []string

	if len(artists) > 0 {
		top := topK(artists, 5)
		names := make([]string, len(top))
		for i, kv := range top {
			names[i] = fmt.Sprintf("%s(%.0f%%)", kv.k, kv.v*100)
		}
		parts = append(parts, "爱听歌手："+strings.Join(names, "、"))
	}

	if len(genres) > 0 {
		top := topK(genres, 5)
		names := make([]string, len(top))
		for i, kv := range top {
			names[i] = fmt.Sprintf("%s(%.0f%%)", kv.k, kv.v*100)
		}
		parts = append(parts, "偏好风格："+strings.Join(names, "、"))
	}

	if len(langs) > 0 {
		top := topK(langs, 2)
		names := make([]string, len(top))
		for i, kv := range top {
			names[i] = fmt.Sprintf("%s(%.0f%%)", kv.k, kv.v*100)
		}
		parts = append(parts, "偏好语言："+strings.Join(names, "、"))
	}

	if len(parts) == 0 {
		return ""
	}
	return "【用户偏好记录】" + strings.Join(parts, "；") + "。选歌时请优先参考这些信息。"
}

type kv struct {
	k string
	v float64
}

func topK(m map[string]float64, k int) []kv {
	var items []kv
	for key, val := range m {
		items = append(items, kv{key, val})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v > items[j].v })
	if len(items) > k {
		items = items[:k]
	}
	return items
}
