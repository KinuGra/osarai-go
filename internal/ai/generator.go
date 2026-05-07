package ai

import "context"

// Question は AI が生成した1問分のデータ。
// json タグは LLM レスポンスの JSON キー名と一致させる必要がある。
// 特に correct_answer のように snake_case を含むフィールドは
// タグなしだと json.Unmarshal でマッチしないため必須。
type Question struct {
	Title         string   `json:"title"`          // 短いトピック名（例: "errors.Is vs errors.As"）
	Category      string   `json:"category"`       // "design" | "language" | "framework"
	Type          string   `json:"type"`           // "choice" | "written"
	Body          string   `json:"body"`           // 問題文
	Choices       []string `json:"choices"`        // 選択肢（Type="choice" の場合のみ）
	CorrectAnswer string   `json:"correct_answer"` // 正解（choice: "A"〜"D", written: 模範解答）
}

// GenerateRequest は問題生成のリクエスト。
type GenerateRequest struct {
	Diff     string // git diff の内容
	Language string // プログラミング言語（例: "Go"）
	FilePath string // 対象ファイルパス（コンテキスト用）
}

// Generator は diff から問題を生成する構造体。
// LLMProvider を内部に持ち、プロンプトのテンプレート適用 + JSON パースを担当する。
type Generator struct {
	provider LLMProvider
}

// NewGenerator は Generator を生成する。
// provider に Gemini の実装やモックを渡す（DI）。
func NewGenerator(provider LLMProvider) *Generator {
	return &Generator{provider: provider}
}

// GenerateQuestions は diff から問題を生成する。
// TODO: 後続 Issue でプロンプトテンプレートを実装する。
func (g *Generator) GenerateQuestions(ctx context.Context, req GenerateRequest) ([]Question, error) {
	panic("not implemented")
}
