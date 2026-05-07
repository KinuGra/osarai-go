package ai

import "context"

// CompletionRequest は LLM への問い合わせリクエスト。
// プロンプト（指示文）とモデル設定をまとめて持つ。
//
// 【なぜ汎用的な型にするのか】
// Generator（問題生成）も Grader（採点）も、最終的には
// 「プロンプトを送って JSON を受け取る」という同じ処理。
// リクエストの型を共通化しておけば、LLMProvider は1つで済む。
type CompletionRequest struct {
	Prompt      string  // LLM に送るプロンプト文字列
	MaxTokens   int     // レスポンスの最大トークン数（0 ならデフォルト）
	Temperature float64 // 0.0〜1.0。低いほど deterministic、高いほどランダム
}

// CompletionResponse は LLM からのレスポンス。
type CompletionResponse struct {
	Content string // LLM が返したテキスト（JSON 文字列が入る想定）
}

// LLMProvider は大規模言語モデルへの問い合わせを抽象化するインターフェース。
//
// 【なぜインターフェースにするのか】
// - 今は Gemini だけだが、将来 OpenAI や Claude に差し替えたくなるかもしれない
// - テスト時にはモック（ダミー）を使って API 呼び出しなしで開発したい
// - core/ パッケージは「Complete が使える何か」としか知らないので、実装を自由に入れ替えられる
//
// 【context.Context とは】
// Go 標準の「キャンセル・タイムアウト」の仕組み。
// 例: ユーザーが Ctrl+C → ctx がキャンセル → API 呼び出しが中断される。
// ctx を第1引数にするのは Go の慣習。
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
