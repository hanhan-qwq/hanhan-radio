package selectsong

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"hanhan-radio/agentruntime/log"
)

// Indexer pre-imports songs from a JSON manifest into the search index.
type Indexer struct {
	store    *IndexStore
	embedder Embedder
}

func NewIndexer(store *IndexStore, embedder Embedder) *Indexer {
	return &Indexer{store: store, embedder: embedder}
}

// Import reads songs from a JSON file, generates embeddings, and writes to the store.
// Existing songs (matched by title + artist) are skipped.
func (ix *Indexer) Import(ctx context.Context, manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var entries []SongEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no songs in manifest")
	}

	// Filter out already-indexed entries.
	type pair struct {
		entry SongEntry
		text  string
	}
	var toImport []pair
	for _, e := range entries {
		existing, err := ix.store.ByTitleArtist(e.Title, e.Artist)
		if err != nil {
			return fmt.Errorf("check existing %s - %s: %w", e.Artist, e.Title, err)
		}
		if existing != nil {
			log.L().Infow("index_skip_exists", "artist", e.Artist, "title", e.Title)
			continue
		}
		toImport = append(toImport, pair{entry: e, text: BuildEmbeddingText(&e)})
	}

	if len(toImport) == 0 {
		log.L().Infow("index_up_to_date", "total", len(entries))
		return nil
	}

	log.L().Infow("index_start", "new", len(toImport), "total", len(entries))

	texts := make([]string, len(toImport))
	for i, p := range toImport {
		texts[i] = p.text
	}

	vecs, err := ix.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	for i, p := range toImport {
		is := &IndexedSong{
			Title:     p.entry.Title,
			Artist:    p.entry.Artist,
			Year:      p.entry.Year,
			Album:     p.entry.Album,
			Genre:     p.entry.Genre,
			Language:  p.entry.Language,
			ExtraInfo: p.entry.ExtraInfo,
			Event:     p.entry.Event,
			Embedding: vecs[i],
		}
		if err := ix.store.Upsert(is); err != nil {
			return fmt.Errorf("upsert %s - %s: %w", p.entry.Artist, p.entry.Title, err)
		}
		log.L().Infow("indexed", "artist", p.entry.Artist, "title", p.entry.Title)
	}

	log.L().Infow("index_done", "count", len(toImport))
	return nil
}
