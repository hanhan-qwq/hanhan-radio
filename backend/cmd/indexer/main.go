package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"hanhan-radio/agentruntime/selectsong"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	ctx := context.Background()

	// Pass empty endpoint — NewEmbedder picks Ollama if OLLAMA_EMBEDDING_MODEL is set,
	// otherwise falls back to Ark with ARK_EMBEDDING_MODEL or ARK_MODEL.
	embedder, err := selectsong.NewEmbedder("")
	if err != nil {
		log.Fatalf("create embedder: %v", err)
	}

	store, err := selectsong.OpenIndexStore("data/song_index.db")
	if err != nil {
		log.Fatalf("open index store: %v", err)
	}

	indexer := selectsong.NewIndexer(store, embedder)

	manifest := "music/songs.json"
	if len(os.Args) > 1 {
		manifest = os.Args[1]
	}

	if err := indexer.Import(ctx, manifest); err != nil {
		log.Fatalf("import: %v", err)
	}

	count, _ := store.Count()
	log.Printf("索引完成，共 %d 首歌曲", count)
}
