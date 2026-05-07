package ai

import "context"

// CompletionRequest は LLM への問い合わせリクエスト。
// SystemPrompt と UserPrompt を分離することで、Gemini / OpenAI / Claude
// いずれの API でもシステム指示とユーザー指示を正しくマッピングできる。
type CompletionRequest struct {
	SystemPrompt string  // 役割・制約の指示（例: "JSON のみで返答せよ"）
	UserPrompt   string  // 実際のタスク指示（diff やユーザーの回答）
	Temperature  float64 // 0.0〜1.0。低いほど deterministic
	MaxTokens    int     // レスポンスの最大トークン数（0 ならデフォルト）
}

// CompletionResponse は LLM からのレスポンス。
type CompletionResponse struct {
	Content string // LLM が返したテキスト（JSON 文字列が入る想定）
}

// LLMProvider は大規模言語モデルへの問い合わせを抽象化するインターフェース。
//
// 責務: API 呼び出しとレスポンスの取得のみ。
// プロンプトテンプレートの適用や JSON パースは generator / grader が担当する。
//
// 新しいプロバイダーを追加する場合は Complete() を実装するだけでよく、
// generator / grader 側は一切変更不要。
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
