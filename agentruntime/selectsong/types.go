package selectsong

import (
	"encoding/gob"

	"github.com/cloudwego/eino/schema"
)

// SelectSongInput is the input for the select_song tool (unchanged — agent-facing).
type SelectSongInput struct {
	Mood     string `json:"mood,omitempty" jsonschema_description:"用户心情，如轻松、愉快、伤感、兴奋"`
	Genre    string `json:"genre,omitempty" jsonschema_description:"音乐风格，如流行、摇滚、爵士、电子、古典"`
	Artist   string `json:"artist,omitempty" jsonschema_description:"指定歌手名称"`
	Language string `json:"language,omitempty" jsonschema_description:"语言偏好，如中文、英文、日文、韩文"`
}

// SelectSongOutput is the output of the select_song tool (unchanged — agent-facing).
type SelectSongOutput struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album,omitempty"`
	AudioURL string `json:"audio_url"`
	Duration int    `json:"duration"`
	Genre    string `json:"genre,omitempty"`
}

// SongEntry is a raw song entry from songs.json.
type SongEntry struct {
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Year      int    `json:"year"`
	Album     string `json:"album"`
	Genre     string `json:"genre"`
	Language  string `json:"language"`
	ExtraInfo string `json:"extra_info"` // reserved for future ad-hoc fields
	Event     string `json:"event"`      // e.g. "圣诞", "情人节"; empty for now
}

// IndexedSong is a song stored in the search index with metadata and embedding.
// Uniqueness is enforced on (title, artist).
type IndexedSong struct {
	ID        uint      `gorm:"primaryKey"`
	Title     string    `gorm:"not null;uniqueIndex:idx_title_artist"`
	Artist    string    `gorm:"not null;uniqueIndex:idx_title_artist"`
	Year      int
	Album     string
	Genre     string    `gorm:"default:'';index"`
	Language  string    `gorm:"default:'';index"`
	ExtraInfo string    // reserved for future ad-hoc fields
	Event     string    `gorm:"default:''"`
	Embedding []float64 `gorm:"type:blob;serializer:gob"`
}

// StructuredQuery is the output of the LLM query-understanding step.
type StructuredQuery struct {
	Mood      string   `json:"mood"`
	Genre     string   `json:"genre"`
	Artist    string   `json:"artist"`
	Language  string   `json:"language"`
	Keywords  []string `json:"keywords"`   // free-text keywords for embedding
	BPMFast   bool     `json:"bpm_fast"`   // true → prefer high BPM
	BPMSlow   bool     `json:"bpm_slow"`   // true → prefer low BPM
}

// SearchCandidate pairs a song with its similarity score.
type SearchCandidate struct {
	Song       IndexedSong
	Similarity float64 // cosine similarity, 0..1
}

// RerankContext carries server-side context injected into the rerank prompt.
// The agent does NOT pass this — it's attached by the server when building the chain.
type RerankContext struct {
	UserProfile string   // user's music taste profile
	RecentPlays []string  // recently played "artist - title", most recent first
}

// RecentPlayItem is a recently played song entry.
type RecentPlayItem struct {
	Artist string
	Title  string
}

// ---- internal chain types ----

// rawQuery carries the original agent-input into the query-understanding step.
type rawQuery struct {
	Input *SelectSongInput
}

// structuredQuery holds the parsed query and the candidate pool.
type structuredQuery struct {
	Input      *SelectSongInput
	Parsed     StructuredQuery
	QueryVec   []float64 // embedding of the query text
}

// rerankInput carries candidates + context to the final LLM rerank step.
type rerankInput struct {
	Input      *SelectSongInput
	Candidates []SearchCandidate
	Messages   []*schema.Message
}

// ConfirmSongInfo is the user-facing interrupt payload shown after song selection.
type ConfirmSongInfo struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Genre    string `json:"genre,omitempty"`
	Language string `json:"language,omitempty"`
}

// selectSongState is the tool's internal state saved in checkpoint during interrupt.
type selectSongState struct {
	Output SelectSongOutput
}

func init() {
	gob.RegisterName("hanhan-sel-selectSongState", &selectSongState{})
	gob.RegisterName("hanhan-sel-ConfirmSongInfo", &ConfirmSongInfo{})
}
