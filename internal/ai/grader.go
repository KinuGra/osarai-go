package ai

import "context"

// GradeRequest は採点リクエスト。
type GradeRequest struct {
	Question    Question // 元の問題
	UserAnswer  string   // ユーザーの回答
	DiffContext string   // 元の diff（文脈として AI に渡す）
}

// GradeResult は AI による採点結果。
// IsCorrect と Score はポインタ型で、採点不能時（記述式 API 失敗など）は nil になる。
type GradeResult struct {
	IsCorrect   *bool    // nil = 採点不能
	Score       *float64 // 0.0〜10.0。nil = 採点不能
	ScoreLabel  string   // 表示用（例: "9/10"）。空文字 = 採点不能
	Explanation string   // 参考書テキスト風の解説（md 形式）
	GradeStatus string   // "graded" | "local_only" | "failed_retryable"
}

// Grader はユーザーの回答を採点する構造体。
type Grader struct {
	provider LLMProvider
}

// NewGrader は Grader を生成する。
func NewGrader(provider LLMProvider) *Grader {
	return &Grader{provider: provider}
}

// GradeAnswer はユーザーの回答を採点し、参考書風の解説を生成する。
//
// フォールバック仕様:
//   - 選択式 + API 成功: 全フィールドを返す（GradeStatus="graded"）
//   - 選択式 + API 失敗: ローカル比較のみ（GradeStatus="local_only"）
//   - 記述式 + API 失敗: IsCorrect=nil（GradeStatus="failed_retryable"）
//
// TODO: 後続 Issue でプロンプトテンプレートとフォールバックロジックを実装する。
func (g *Grader) GradeAnswer(ctx context.Context, req GradeRequest) (GradeResult, error) {
	panic("not implemented")
}
