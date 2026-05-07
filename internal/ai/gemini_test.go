package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KinuGra/osarai-go/internal/apperror"
)

// mockGeminiServer はテスト用の Gemini API モックサーバーを起動する。
// handler で任意のレスポンスを返せる。
func mockGeminiServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GeminiProvider) {
	t.Helper()
	srv := httptest.NewServer(handler)

	// GeminiProvider のエンドポイントをモックサーバーに向ける
	provider := &GeminiProvider{
		apiKey:     "test-api-key",
		model:      "gemini-2.5-flash",
		httpClient: &http.Client{},
	}

	// テスト用にベース URL をモックサーバーに差し替える
	// （geminiAPIBaseURL 定数は上書きできないため、直接 URL を組み立てる別メソッドを使う）
	// ここでは httpClient の Transport を差し替えて全リクエストをモックへリダイレクトする
	provider.httpClient.Transport = &mockTransport{mockURL: srv.URL}

	return srv, provider
}

// mockTransport はリクエスト先を本番 Gemini API からモックサーバーへ書き換える。
type mockTransport struct {
	mockURL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = t.mockURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(req2)
}

func TestNewGeminiProvider_EmptyAPIKey(t *testing.T) {
	_, err := NewGeminiProvider("", "")
	if err == nil {
		t.Fatal("空の API キーでエラーが返らなかった")
	}
	if err != apperror.ErrAPIKeyMissing {
		t.Errorf("期待: apperror.ErrAPIKeyMissing, 実際: %v", err)
	}
}

func TestNewGeminiProvider_DefaultModel(t *testing.T) {
	p, err := NewGeminiProvider("test-key", "")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if p.model != defaultGeminiModel {
		t.Errorf("期待: %s, 実際: %s", defaultGeminiModel, p.model)
	}
}

func TestGeminiProvider_Complete_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディの検証
		var reqBody geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("リクエストボディのパースに失敗: %v", err)
		}
		if len(reqBody.Contents) == 0 {
			t.Error("contents が空")
		}

		// 正常レスポンスを返す
		resp := geminiResponse{}
		resp.Candidates = []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		}{
			{
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{{Text: "Hello from Gemini!"}},
				},
				FinishReason: "STOP",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	srv, provider := mockGeminiServer(t, handler)
	defer srv.Close()

	result, err := provider.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "You are a helpful assistant.",
		UserPrompt:   "Say hello",
		Temperature:  0.7,
	})
	if err != nil {
		t.Fatalf("Complete エラー: %v", err)
	}
	if result.Content != "Hello from Gemini!" {
		t.Errorf("期待: 'Hello from Gemini!', 実際: %q", result.Content)
	}
}

func TestGeminiProvider_Complete_RateLimit(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"RESOURCE_EXHAUSTED","status":"RESOURCE_EXHAUSTED"}}`))
	}

	srv, provider := mockGeminiServer(t, handler)
	defer srv.Close()

	_, err := provider.Complete(context.Background(), CompletionRequest{
		UserPrompt: "test",
	})
	if err == nil {
		t.Fatal("429 エラーが返らなかった")
	}
	if err != apperror.ErrRateLimit {
		t.Errorf("期待: apperror.ErrRateLimit, 実際: %v", err)
	}
}

func TestGeminiProvider_Complete_EmptyResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// candidates が空のレスポンス
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}

	srv, provider := mockGeminiServer(t, handler)
	defer srv.Close()

	_, err := provider.Complete(context.Background(), CompletionRequest{
		UserPrompt: "test",
	})
	if err == nil {
		t.Fatal("空レスポンスでエラーが返らなかった")
	}
}

func TestGeminiProvider_Complete_SystemPromptOmitted(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var reqBody geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("パースエラー: %v", err)
		}
		// SystemPrompt が空のとき systemInstruction は送らない
		if reqBody.SystemInstruction != nil {
			t.Error("SystemPrompt が空なのに systemInstruction が送られた")
		}

		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{
					"parts": []map[string]interface{}{{"text": "ok"}},
				}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	srv, provider := mockGeminiServer(t, handler)
	defer srv.Close()

	_, err := provider.Complete(context.Background(), CompletionRequest{
		UserPrompt: "test",
		// SystemPrompt は空
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
}
