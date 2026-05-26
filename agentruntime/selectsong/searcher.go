package selectsong

import (
	"context"
	"math"
	"math/rand"
	"sort"
)

const (
	searchTopK       = 20
	exploreRatio     = 0.15  // fraction of results perturbed for exploration
	minCandidatesRerank = 5
)

// Searcher performs hybrid search: embedding similarity + metadata filter + exploration.
type Searcher struct {
	store    *IndexStore
	embedder Embedder
}

func NewSearcher(store *IndexStore, embedder Embedder) *Searcher {
	return &Searcher{store: store, embedder: embedder}
}

// Search embeds the query text, computes cosine similarity against all indexed songs,
// applies optional metadata filtering, injects exploration perturbation, and returns top-K.
func (sr *Searcher) Search(ctx context.Context, queryText string) ([]SearchCandidate, error) {
	songs, err := sr.store.All()
	if err != nil {
		return nil, err
	}
	if len(songs) == 0 {
		return nil, nil
	}

	vecs, err := sr.embedder.EmbedStrings(ctx, []string{queryText})
	if err != nil {
		return nil, err
	}
	queryVec := vecs[0]

	candidates := make([]SearchCandidate, len(songs))
	for i, s := range songs {
		candidates[i] = SearchCandidate{
			Song:       s,
			Similarity: cosine(queryVec, s.Embedding),
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Similarity > candidates[j].Similarity
	})

	// Exploration: randomly swap ~15% of the ranked candidates with
	// songs from outside the top-K to prevent cold-song starvation.
	k := searchTopK
	if k > len(candidates) {
		k = len(candidates)
	}
	swapCount := int(float64(k) * exploreRatio)
	if swapCount > 0 && len(candidates) > k {
		for i := 0; i < swapCount; i++ {
			posInTop := k - 1 - i                              // swap from bottom of top-K
			posOutside := k + rand.Intn(len(candidates)-k)     // pick from outside top-K
			candidates[posInTop], candidates[posOutside] = candidates[posOutside], candidates[posInTop]
		}
	}

	return candidates[:k], nil
}

func (sr *Searcher) QueryVec(ctx context.Context, queryText string) ([]float64, error) {
	vecs, err := sr.embedder.EmbedStrings(ctx, []string{queryText})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < minLen; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
