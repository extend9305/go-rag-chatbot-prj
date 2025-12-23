package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"example.com/hello/vector"
)

type Service struct {
	apiURL string
	model  string
	client *http.Client
}

// EmbeddingRequest represents the request to the embedding API
type RerankRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type RerankResponse struct {
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason,omitempty"`
	Context    []int  `json:"context,omitempty"`
}

// Rerank 결과 구조체
type RerankResult struct {
	Results []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	} `json:"results"`
}

type RankedDocument struct {
	Index   int     `json:"index"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// NewService creates a new embedding service
func NewService(apiURL, model string) *Service {
	return &Service{
		apiURL: apiURL,
		model:  model,
		client: &http.Client{},
	}
}

// 질문과 document로 유사도 리스트를 뽑는다.
func (s *Service) Rerank(ctx context.Context, content string, documents []vector.Document) ([]RankedDocument, error) {
	var results []RankedDocument

	prompt := s.buildPrompt(content, documents)

	reqData := RerankRequest{
		Model:  s.model,
		Prompt: prompt,
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
	// 원시 응답 읽기
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 1단계: 외부 JSON 파싱
	var response RerankResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Printf("Response field (escaped JSON): |%s|", response.Response)
	cleanResponse := strings.TrimSpace(response.Response)
	cleanResponse = strings.Trim(cleanResponse, "\"'`") // 따옴표 제거

	log.Printf("Response field (escaped JSON): |%s|", cleanResponse)
	// 2단계: response 필드 내부의 이스케이프된 JSON 파싱
	var rerankResult RerankResult
	if err := json.Unmarshal([]byte(cleanResponse), &rerankResult); err != nil {
		return nil, fmt.Errorf("failed to parse rerank result: %w, response was: %s", err, response.Response)
	}

	for _, rerank := range rerankResult.Results {
		log.Println(rerank)
		if rerank.Score > 0.6 {
			document := RankedDocument{
				Index:   rerank.Index,
				Content: documents[rerank.Index].Content,
				Score:   rerank.Score,
			}
			log.Printf("Selected document - Content: %s, Score: %.3f\n", document.Content, document.Score)
			results = append(results, document)
		}
	}

	return results, nil
}

func toStringSlice(docs []vector.Document) []string {
	result := make([]string, len(docs))
	for i, doc := range docs {
		result[i] = doc.Content
	}
	return result
}

// ollama run llama3-3b-rerank <<'EOF'
// 너는 rerank 전용 모델이다.
//
// 규칙:
// - 출력은 반드시 JSON 하나만 출력한다
// - JSON 출력이 끝나면 즉시 종료한다
// - 문서 목록에 있는 모든 문서에 대해 점수를 매겨야 한다
// - 문서 개수와 results 배열의 길이는 반드시 같아야 한다
// - JSON 뒤에 어떤 텍스트도 출력하지 않는다
// - 설명, 문장, 개행을 절대 추가하지 않는다
//
// 출력 형식:
// {
// "results": [
// {"index": number, "score": number}
// ]
// }
//
// 질문:
// 한국의 수도는 어딜까?
//
// 문서 목록:
// [0] 한국의 수도는 서울입니다.
// [1] 한국의 수도는 서울이다.
// [2] 테스트입니다.
// EOF
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxChars int) string {
	if len(s) > maxChars {
		return s[:maxChars]
	}
	return s
}
func (s *Service) buildPrompt(query string, documents []vector.Document) string {
	const (
		maxDocs     = 3   // LLM rerank 대상 최대 개수
		maxDocChars = 200 // 문서당 최대 길이
	)

	// 1. 문서 수 제한 (Top-K)
	documents = documents[:min(len(documents), maxDocs)]

	var sb strings.Builder

	// 2. 시스템 지시문 (최소화)
	sb.WriteString("You are a reranking system.\n")
	sb.WriteString("Score each document for relevance to the query.\n\n")

	// 3. 강제 규칙
	sb.WriteString("RULES:\n")
	sb.WriteString("OUTPUT ONLY VALID JSON. NO EXTRA TEXT.\n\n")
	sb.WriteString("- No markdown, no extra text\n")
	sb.WriteString(fmt.Sprintf("- Exactly %d results\n", len(documents)))
	sb.WriteString("- Score range: 0.0 to 1.0\n")
	sb.WriteString("- Consider both semantic relevance AND vector distance\n")
	sb.WriteString("- Start with { and end with }\n\n")

	// 4. Query
	sb.WriteString("Query:\n")
	sb.WriteString(query)
	sb.WriteString("\n\n")

	// 5. Documents (truncate + 압축 포맷)
	sb.WriteString("Documents:\n")
	for i, doc := range documents {
		content := truncate(doc.Content, maxDocChars)
		sb.WriteString(fmt.Sprintf("D%d (distance: %.3f): %s\n", i, 1-doc.Distance, content))
	}

	// 6. 출력 포맷 명시
	sb.WriteString("\nOutput JSON format:\n")
	sb.WriteString(`{"results":[`)
	for i := range documents {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`{"index":%d,"score":0.0}`, i))
	}
	sb.WriteString(`]}`)
	return sb.String()
}

// FastRerank - 규칙 기반 빠른 reranking
func (s *Service) FastRerank(ctx context.Context, query string, documents []vector.Document) ([]RankedDocument, error) {
	queryTokens := tokenize(query)

	var ranked []RankedDocument

	for _, doc := range documents {
		// 1. 벡터 유사도 (이미 계산됨)
		vectorScore := 1.0 - doc.Distance

		// 2. 키워드 매칭 점수
		keywordScore := calculateKeywordMatch(queryTokens, doc.Content)

		// 3. 문서 길이 패널티 (너무 긴 문서는 점수 낮춤)
		lengthPenalty := calculateLengthPenalty(doc.Content)

		// 최종 점수 = 벡터 70% + 키워드 20% + 길이 10%
		finalScore := vectorScore*0.5 + keywordScore*0.4 + lengthPenalty*0.1

		ranked = append(ranked, RankedDocument{
			Content: doc.Content,
			Score:   finalScore,
		})
	}

	// 점수 기준 정렬
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	// 상위 N개만 반환
	if len(ranked) > 3 {
		ranked = ranked[:3]
	}

	return ranked, nil
}

// tokenize - 간단한 토크나이저
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// 공백과 특수문자로 분리
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '가' && r <= '힣') || (r >= '0' && r <= '9'))
	})
	return tokens
}

// calculateKeywordMatch - 키워드 매칭 점수
func calculateKeywordMatch(queryTokens []string, docContent string) float64 {
	if len(queryTokens) == 0 {
		return 0.5
	}

	docTokens := tokenize(docContent)
	docTokenSet := make(map[string]bool)
	for _, token := range docTokens {
		docTokenSet[token] = true
	}

	matchCount := 0
	for _, qToken := range queryTokens {
		if docTokenSet[qToken] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(queryTokens))
}

// calculateLengthPenalty - 문서 길이 패널티
func calculateLengthPenalty(content string) float64 {
	length := len([]rune(content))

	// 최적 길이: 50-200자
	if length >= 50 && length <= 200 {
		return 1.0
	} else if length < 50 {
		return float64(length) / 50.0
	} else {
		// 200자 이상은 패널티
		return 1.0 - (float64(length-200) / 1000.0)
	}
}
