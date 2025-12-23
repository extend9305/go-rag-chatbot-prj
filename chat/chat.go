package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"example.com/hello/reranker"
)

type Service struct {
	apiURL string
	model  string
	client *http.Client
}

// ChatRequest represents the request to the chat API
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents the response from the chat API
type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

// NewService creates a new embedding service
func NewService(apiURL, model string) *Service {
	return &Service{
		apiURL: apiURL,
		model:  model,
		client: &http.Client{},
	}
}

// Chat sends a message to the chat API with context documents
func (s *Service) Chat(ctx context.Context, userQuestion string, contextDocuments []reranker.RankedDocument) (string, error) {
	var sb strings.Builder

	sb.WriteString("당신은 주어진 모든 참고 문서들을 종합하여 질문에 친절하게 안내하는 챗봇입니다.\n\n")
	sb.WriteString("중요 규칙:\n")
	sb.WriteString("1. 모든 참고 문서를 종합해 자연스러운 한두 문장으로 답변하세요.\n")
	sb.WriteString("2. 필요한 정보만 간결하고 부드러운 말투로 안내하세요.\n\n")
	sb.WriteString("3. 정보가 없으면 정중히 없다고 답하세요.\n\n")

	if len(contextDocuments) > 0 {
		sb.WriteString(fmt.Sprintf("=== 참고 문서 (총 %d개) ===\n", len(contextDocuments)))
		for i, doc := range contextDocuments {
			sb.WriteString(fmt.Sprintf("\n[문서 %d] (관련도: %.2f)\n", i+1, doc.Score))
			sb.WriteString(doc.Content)
			sb.WriteString("\n")
		}
		sb.WriteString("\n=== 모든 문서를 검토한 후 합해서 답변하세요 ===\n")
	} else {
		sb.WriteString("참고할 문서가 없습니다.\n")
	}

	systemPrompt := sb.String()

	// 메시지 구성
	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userQuestion,
		},
	}
	// 요청 데이터 생성
	reqData := ChatRequest{
		Model:    s.model,
		Messages: messages,
		Stream:   false,
	}
	fmt.Println(reqData)
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ngrok-skip-browser-warning", "true")

	// 요청 전송
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}
