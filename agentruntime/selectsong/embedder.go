package selectsong

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// Embedder generates text embeddings from a backend (Ark or local Ollama).
type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

// NewEmbedder picks the embedding backend:
// If OLLAMA_EMBEDDING_MODEL env is set, uses local Ollama.
// Otherwise falls back to Ark with the given endpoint.
func NewEmbedder(arkEndpoint string) (Embedder, error) {
	ollamaModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if ollamaModel != "" {
		return newOllamaEmbedder(ollamaModel)
	}
	return newArkEmbedder(arkEndpoint)
}

// ——— Ark ———————————————————————————————————————————————————

type arkEmbedder struct {
	client   *arkruntime.Client
	endpoint string
}

func newArkEmbedder(endpoint string) (Embedder, error) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY not set")
	}
	client := arkruntime.NewClientWithApiKey(apiKey,
		arkruntime.WithBaseUrl("https://ark.cn-beijing.volces.com/api/v3"),
		arkruntime.WithRegion("cn-beijing"),
	)
	return &arkEmbedder{client: client, endpoint: endpoint}, nil
}

func (e *arkEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	req := model.EmbeddingRequestStrings{
		Input:          texts,
		Model:          e.endpoint,
		EncodingFormat: model.EmbeddingEncodingFormatFloat,
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ark embeddings: %w", err)
	}

	out := make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		v := make([]float64, len(d.Embedding))
		for j, f := range d.Embedding {
			v[j] = float64(f)
		}
		out[i] = v
	}
	return out, nil
}

// ——— Ollama —————————————————————————————————————————————————

type ollamaEmbedder struct {
	model   string
	baseURL string
	client  *http.Client
}

func newOllamaEmbedder(model string) (Embedder, error) {
	return &ollamaEmbedder{
		model:   model,
		baseURL: "http://localhost:11434",
		client:  &http.Client{},
	}, nil
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (e *ollamaEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	body, _ := json.Marshal(ollamaEmbedRequest{
		Model: e.model,
		Input: texts,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama embed: status %d", resp.StatusCode)
	}

	var out ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}

	return out.Embeddings, nil
}

// BuildEmbeddingText constructs the text to embed from song metadata.
func BuildEmbeddingText(s *SongEntry) string {
	parts := []string{s.Title, s.Artist}
	if s.Album != "" {
		parts = append(parts, s.Album)
	}
	if s.Genre != "" {
		parts = append(parts, s.Genre)
	}
	if s.Language != "" {
		parts = append(parts, s.Language)
	}
	if s.ExtraInfo != "" {
		parts = append(parts, s.ExtraInfo)
	}
	return strings.Join(parts, " ")
}
