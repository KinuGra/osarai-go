package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/KinuGra/osarai-go/internal/apperror"
)

const (
	defaultGeminiModel     = "gemini-2.5-flash"
	geminiAPIBaseURL       = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultHTTPTimeout     = 60 * time.Second
	defaultMaxOutputTokens = 8192
)

// GeminiProvider は Google Gemini API の LLMProvider 実装。
// net/http で REST API を直接呼び出すことで、外部 SDK への依存を排除し
// シングルバイナリ配布を容易にする。
type GeminiProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGeminiProvider は GeminiProvider を生成する。
// model が空文字の場合は defaultGeminiModel を使用する。
// apiKey が空の場合は apperror.ErrAPIKeyMissing を返す。
func NewGeminiProvider(apiKey, model string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, apperror.ErrAPIKeyMissing
	}
	if model == "" {
		model = defaultGeminiModel
	}
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}, nil
}

// ---- Gemini REST API のリクエスト / レスポンス型 ----

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

type geminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Complete は Gemini API にプロンプトを送り、テキストレスポンスを返す。
func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	body, err := p.buildRequestBody(req)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("リクエスト構築エラー: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIBaseURL, p.model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("HTTP リクエスト生成エラー: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("Gemini API 通信エラー: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("レスポンス読み取りエラー: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, p.parseAPIError(resp.StatusCode, respBody)
	}

	return p.parseResponse(respBody)
}

// buildRequestBody は CompletionRequest を Gemini REST API 用 JSON に変換する。
func (p *GeminiProvider) buildRequestBody(req CompletionRequest) ([]byte, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxOutputTokens
	}

	gemReq := geminiRequest{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: req.UserPrompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: maxTokens,
		},
	}

	if req.SystemPrompt != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	return json.Marshal(gemReq)
}

// parseResponse は Gemini API の成功レスポンスからテキストを抽出する。
func (p *GeminiProvider) parseResponse(body []byte) (CompletionResponse, error) {
	var gemResp geminiResponse
	if err := json.Unmarshal(body, &gemResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("レスポンス JSON パースエラー: %w", err)
	}

	if len(gemResp.Candidates) == 0 ||
		len(gemResp.Candidates[0].Content.Parts) == 0 {
		return CompletionResponse{}, apperror.ErrNoQuestionsGenerated
	}

	candidate := gemResp.Candidates[0]
	text := candidate.Content.Parts[0].Text
	if text == "" {
		return CompletionResponse{}, apperror.ErrNoQuestionsGenerated
	}

	// 出力がトークン上限で途中切れになった場合は明示的なエラーを返す
	if candidate.FinishReason == "MAX_TOKENS" {
		return CompletionResponse{}, fmt.Errorf(
			"Gemini API の出力がトークン上限に達したため応答が不完全です（FinishReason: MAX_TOKENS）。"+
				"MaxOutputTokens を増やすか、プロンプトを短くしてください",
		)
	}

	return CompletionResponse{Content: text}, nil
}

// parseAPIError は HTTP エラーレスポンスを apperror に変換する。
func (p *GeminiProvider) parseAPIError(statusCode int, body []byte) error {
	if statusCode == http.StatusTooManyRequests {
		return apperror.ErrRateLimit
	}

	var errResp geminiErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("Gemini API エラー (HTTP %d): %s", statusCode, errResp.Error.Message)
	}

	return fmt.Errorf("Gemini API エラー (HTTP %d)", statusCode)
}
