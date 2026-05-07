package ai

import "context"

// GradeRequest は採点リクエスト。
type GradeRequest struct {
	Question    Question // 元の問題
	UserAnswer  string   // ユーザーの回答
	DiffContext string   // 元の diff（文脈として AI に渡す）
}

// GradeResult は AI による採点結果。
// LLM レスポンスの JSON を json.Unmarshal するため、snake_case フィールドには
// タグが必須（タグなしだと is_correct → IsCorrect のマッピングが失敗する）。
// IsCorrect と Score はポインタ型で、採点不能時（記述式 API 失敗など）は nil になる。
type GradeResult struct {
	IsCorrect   *bool    `json:"is_correct"`          // nil = 採点不能
	Score       *float64 `json:"score"`               // 0.0〜10.0。nil = 採点不能
	ScoreLabel  string   `json:"score_label"`         // 表示用（例: "9/10"）。空文字 = 採点不能
	Explanation string   `json:"explanation"`         // 参考書テキスト風の解説（md 形式）
	GradeStatus string   `json:"grade_status"`        // "graded" | "local_only" | "failed_retryable"
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
