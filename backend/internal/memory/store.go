package memory

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PlayRecord is a single song play in history.
type PlayRecord struct {
	ID         uint   `gorm:"primaryKey"`
	SongTitle  string `gorm:"not null;index"`
	SongArtist string `gorm:"not null;index"`
	SongGenre  string `gorm:"default:''"`
	Language   string `gorm:"default:''"`
	UserPrompt string `gorm:"not null"`
	SessionID  string `gorm:"not null;index"`
	CreatedAt  int64  `gorm:"autoCreateTime:milli"`
}

// ProfileEntry is a single key-value preference score.
type ProfileEntry struct {
	Key         string  `gorm:"primaryKey"`
	Value       float64 `gorm:"not null"`
	LastUpdated int64   `gorm:"autoUpdateTime:milli"`
}

// ConversationTurn is a single message in a conversation session.
type ConversationTurn struct {
	ID        uint   `gorm:"primaryKey"`
	SessionID string `gorm:"not null;index:idx_session_created"`
	Role      string `gorm:"not null"` // "user" | "assistant"
	Content   string `gorm:"not null"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_session_created"`
}

// MemoryFact is a natural-language fact in the user's long-term profile.
type MemoryFact struct {
	ID         uint    `gorm:"primaryKey"`
	Category   string  `gorm:"not null;index:idx_fact_category"`
	Content    string  `gorm:"not null"`
	Confidence float64 `gorm:"default:0.5"`
	Source     string  `gorm:"default:''"`
	CreatedAt  int64   `gorm:"autoCreateTime:milli"`
	ExpiredAt  int64   `gorm:"default:0;index:idx_fact_expired"` // 0 = active
}

// Store wraps the GORM DB for all memory operations.
type Store struct {
	DB *gorm.DB
}

// Open initializes the SQLite database and runs auto-migration.
func Open(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&PlayRecord{}, &ProfileEntry{}, &ConversationTurn{}, &MemoryFact{}); err != nil {
		return nil, err
	}

	return &Store{DB: db}, nil
}
