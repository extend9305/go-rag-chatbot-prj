package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Service struct {
	apiURL string
	model  string
	client *http.Client
}

// EmbeddingRequest represents the request to the embedding API
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// EmbeddingResponse represents the response from the embedding API
type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// NewService creates a new embedding service
func NewService(apiURL, model string) *Service {
	return &Service{
		apiURL: apiURL,
		model:  model,
		client: &http.Client{},
	}
}

// GenerateEmbedding generates embedding vector from text
func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// 요청 데이터 생성
	reqData := EmbeddingRequest{
		Model:  s.model,
		Prompt: text,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ngrok-skip-browser-warning", "true")

	// 요청 전송
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// float64 -> float32 변환
	embedding32 := make([]float32, len(embResp.Embedding))
	for i, v := range embResp.Embedding {
		embedding32[i] = float32(v)
	}

	return embedding32, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func (s *Service) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := s.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}
