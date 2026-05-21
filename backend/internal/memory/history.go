package memory

import (
	"fmt"
	"strings"
)

const recentPlayLimit = 30

// InsertPlay records a song play.
func (s *Store) InsertPlay(r *PlayRecord) error {
	return s.DB.Create(r).Error
}

// RecentPlays returns the most recent N play records for dedup.
func (s *Store) RecentPlays() ([]PlayRecord, error) {
	var records []PlayRecord
	err := s.DB.Order("created_at DESC").Limit(recentPlayLimit).Find(&records).Error
	return records, err
}

// SearchPlays searches play history by keyword (title or artist) within recent days.
func (s *Store) SearchPlays(keyword string, days int) ([]PlayRecord, error) {
	var records []PlayRecord
	q := s.DB.Order("created_at DESC").Limit(10)
	if days > 0 {
		q = q.Where("created_at > strftime('%s','now','-' || ? || ' days') * 1000", fmt.Sprint(days))
	}
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where("LOWER(song_title) LIKE ? OR LOWER(song_artist) LIKE ?", like, like)
	}
	err := q.Find(&records).Error
	return records, err
}
