package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"hanhan-radio/agentruntime/log"
)

const defaultBaseURL = "https://api.xiaomimimo.com/v1"

// Config holds the MimoTTS client configuration.
type Config struct {
	APIKey string
	Model  string // default "mimo-v2.5-tts"
	Voice  string // default "冰糖"
	Format string // default "wav"
	Style  string // style instruction placed in user message
}

// Client is a MimoTTS HTTP client.
type Client struct {
	cfg    Config
	http   *http.Client
	baseURL string
}

// NewClient creates a new MimoTTS client.
func NewClient(cfg Config) *Client {
	if cfg.Model == "" {
		cfg.Model = "mimo-v2.5-tts"
	}
	if cfg.Voice == "" {
		cfg.Voice = "冰糖"
	}
	if cfg.Format == "" {
		cfg.Format = "wav"
	}
	if cfg.Style == "" {
		cfg.Style = "温暖亲切的女声，娓娓道来，语速适中偏慢，像深夜电台 DJ 在陪伴听众，声音温柔而有磁性"
	}

	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: 60 * time.Second},
		baseURL: defaultBaseURL,
	}
}

// Synthesize converts text to speech, returning the raw audio bytes (WAV).
func (c *Client) Synthesize(ctx context.Context, text string) ([]byte, error) {
	start := time.Now()

	body := chatRequest{
		Model: c.cfg.Model,
		Messages: []message{
			{Role: "user", Content: c.cfg.Style},
			{Role: "assistant", Content: text},
		},
		Audio: audioParams{
			Format: c.cfg.Format,
			Voice:  c.cfg.Voice,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("api-key", c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	elapsed := time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		log.L().Errorw("tts_api_error", "status", resp.StatusCode, "latency_ms", elapsed, "body", string(respBytes))
		return nil, fmt.Errorf("tts api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBytes, &cr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}

	audioData := cr.Choices[0].Message.Audio.Data
	if audioData == "" {
		return nil, fmt.Errorf("empty audio data in response")
	}

	raw, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return nil, fmt.Errorf("decode base64 audio: %w", err)
	}

	log.L().Infow("tts_api_done", "text_len", len(text), "audio_bytes", len(raw), "latency_ms", elapsed)

	return raw, nil
}

type chatRequest struct {
	Model    string       `json:"model"`
	Messages []message    `json:"messages"`
	Audio    audioParams  `json:"audio"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type audioParams struct {
	Format string `json:"format"`
	Voice  string `json:"voice"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Audio struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
}
