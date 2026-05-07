package ai

import (
	"context"
	"os"
	"testing"
)

// TestGeminiProvider_Integration は実際の Gemini API に接続して疎通確認を行う。
// GEMINI_API_KEY 環境変数が未設定の場合はスキップする。
func TestGeminiProvider_Integration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY が未設定のためスキップ")
	}

	provider, err := NewGeminiProvider(apiKey, "")
	if err != nil {
		t.Fatalf("NewGeminiProvider エラー: %v", err)
	}

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "あなたは簡潔に回答するアシスタントです。",
		UserPrompt:   `次の JSON のみを返してください（他のテキストは不要）: {"message": "hello"}`,
		Temperature:  0.1,
		MaxTokens:    64,
	})
	if err != nil {
		t.Fatalf("Complete エラー: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("レスポンスが空")
	}

	t.Logf("Gemini API レスポンス: %s", resp.Content)
	t.Log("✅ Gemini API 疎通確認 OK")
}
