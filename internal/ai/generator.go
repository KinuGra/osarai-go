package ai

import "context"

// QuestionType は問題の種類。
type QuestionType string

const (
	QuestionTypeChoice  QuestionType = "choice"  // 選択式（4択）
	QuestionTypeWritten QuestionType = "written" // 記述式
)

// Question は AI が生成した1問分のデータ。
// JSON タグ（`json:"..."`）は AI のレスポンス JSON → Go 構造体の変換に使う。
type Question struct {
	Title        string       `json:"title"`         // 問題タイトル（例: "スライスの容量について"）
	Body         string       `json:"body"`          // 問題文
	QuestionType QuestionType `json:"question_type"` // "choice" or "written"
	Choices      []string     `json:"choices"`       // 選択肢（choice の場合のみ）
	Answer       string       `json:"answer"`        // 正解（choice: "A"〜"D", written: 模範解答）
	Explanation  string       `json:"explanation"`   // 出題意図の簡単な説明
}

// GenerateRequest は問題生成のリクエスト。
type GenerateRequest struct {
	Diff         string // git diff の内容
	NumQuestions int    // 生成する問題数
}

// GenerateResult は問題生成の結果。
type GenerateResult struct {
	Questions []Question `json:"questions"`
	RawJSON   string     // AI の生レスポンス（デバッグ用）
}

// Generator は diff から問題を生成する構造体。
// LLMProvider を内部に持ち、プロンプトのテンプレート適用 + JSON パースを担当。
type Generator struct {
	provider LLMProvider
}

// NewGenerator は Generator を生成する。
// provider に Gemini の実装やモックを渡す（DI: 依存性注入）。
func NewGenerator(provider LLMProvider) *Generator {
	return &Generator{provider: provider}
}

// Generate は diff から問題を生成する。
// TODO: Issue #5 完了後、機能 A の担当者が実装する。
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	panic("not implemented") // スタブ: まだ中身はない
}
