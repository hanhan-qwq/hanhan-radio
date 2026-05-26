package selectsong

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IndexStore persists indexed songs with embeddings.
type IndexStore struct {
	DB *gorm.DB
}

func OpenIndexStore(path string) (*IndexStore, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&IndexedSong{}); err != nil {
		return nil, err
	}
	return &IndexStore{DB: db}, nil
}

// Upsert inserts a song or updates it if the same (title, artist) already exists.
func (s *IndexStore) Upsert(song *IndexedSong) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "title"}, {Name: "artist"}},
		UpdateAll: true,
	}).Create(song).Error
}

// All returns every indexed song.
func (s *IndexStore) All() ([]IndexedSong, error) {
	var songs []IndexedSong
	err := s.DB.Find(&songs).Error
	return songs, err
}

// Count returns the number of indexed songs.
func (s *IndexStore) Count() (int64, error) {
	var c int64
	err := s.DB.Model(&IndexedSong{}).Count(&c).Error
	return c, err
}

// ByTitleArtist returns a song by title + artist, or nil.
func (s *IndexStore) ByTitleArtist(title, artist string) (*IndexedSong, error) {
	var song IndexedSong
	err := s.DB.Where("title = ? AND artist = ?", title, artist).First(&song).Error
	if err != nil {
		return nil, nil
	}
	return &song, nil
}
