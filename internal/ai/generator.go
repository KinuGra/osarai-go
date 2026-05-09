package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	_ "embed"

	"github.com/KinuGra/osarai-go/internal/apperror"
)

// QuestionType は問題の種類。
type QuestionType string

const (
	QuestionTypeChoice  QuestionType = "choice"  // 選択式（4〜5択）
	QuestionTypeWritten QuestionType = "written" // 記述式
)

// QuestionCategory は問題のカテゴリ。MVP の3カテゴリを定義。
// v2 以降で DesignPattern / Pitfall / Fundamentals を追加予定。
type QuestionCategory string

const (
	QuestionCategoryDesign    QuestionCategory = "design"    // 設計判断
	QuestionCategoryLanguage  QuestionCategory = "language"  // 言語知識
	QuestionCategoryFramework QuestionCategory = "framework" // FW・ライブラリ
)

// Question は AI が生成した1問分のデータ。
// json タグは LLM レスポンスの JSON キー名と一致させる必要がある。
// 特に correct_answer のように snake_case を含むフィールドは
// タグなしだと json.Unmarshal でマッチしないため必須。
type Question struct {
	Title         string           `json:"title"`          // 短いトピック名（例: "errors.Is vs errors.As"）
	Category      QuestionCategory `json:"category"`       // 問題カテゴリ
	Type          QuestionType     `json:"type"`           // 問題種別
	Body          string           `json:"body"`           // 問題文
	Choices       []string         `json:"choices"`        // 選択肢（Type=choice の場合のみ）
	CorrectAnswer string           `json:"correct_answer"` // 正解（choice: "A"〜"D", written: 模範解答）
	Explanation   string           `json:"explanation"`    // 参考書テキスト風の解説（問題生成時に AI が同時出力）
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

//go:embed prompts/generate_questions.tmpl
var generateQuestionsTmplSrc string

var generateQuestionsTmpl = template.Must(
	template.New("generate_questions").Parse(generateQuestionsTmplSrc),
)

const generateSystemPrompt = "あなたはソフトウェアエンジニアの教育専門家です。\n" +
	"git diff を読み、開発者の理解度チェック問題を JSON 配列で生成します。\n" +
	"返答は純粋な JSON 配列のみにしてください。コードブロック（```）、説明文、前置き、後書きは一切不要です。\n" +
	"最初の文字は \"[\" でなければなりません。\n" +
	"JSON 文字列内でダブルクォートを使う場合は必ず \\\" とエスケープしてください。\n" +
	"コード内の識別子はシングルクォート（'）またはバッククォート（`）で囲み、ダブルクォートは使わないでください。"

// GenerateQuestions は diff から問題を生成する。
// テンプレートを適用してプロンプトを構築し、LLM に投げて JSON をパースして返す。
func (g *Generator) GenerateQuestions(ctx context.Context, req GenerateRequest) ([]Question, error) {
	var buf bytes.Buffer
	if err := generateQuestionsTmpl.Execute(&buf, req); err != nil {
		return nil, fmt.Errorf("プロンプトテンプレートの適用に失敗: %w", err)
	}

	resp, err := g.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: generateSystemPrompt,
		UserPrompt:   buf.String(),
		Temperature:  0.7,
		MaxTokens:    8192,
	})
	if err != nil {
		return nil, fmt.Errorf("問題生成 API 呼び出し失敗: %w", err)
	}

	content := cleanJSONResponse(resp.Content)

	var questions []Question
	if err := json.Unmarshal([]byte(content), &questions); err != nil {
		return nil, fmt.Errorf("問題 JSON のパースに失敗: %w\nレスポンス（先頭200文字）: %.200s", err, content)
	}

	if len(questions) == 0 {
		return nil, apperror.ErrNoQuestionsGenerated
	}

	for i := range questions {
		questions[i].Category = normalizeCategory(questions[i].Category)
	}

	return questions, nil
}

// normalizeCategory は DB の CHECK 制約外のカテゴリを最も近い有効値に丸める。
func normalizeCategory(cat QuestionCategory) QuestionCategory {
	switch cat {
	case QuestionCategoryDesign, QuestionCategoryLanguage, QuestionCategoryFramework:
		return cat
	default:
		return QuestionCategoryDesign
	}
}

// cleanJSONResponse は LLM レスポンスから JSON 部分のみを抽出する。
// Markdown コードブロック・改行の有無・前後の説明文に依存しないよう、
// 最初の '[' or '{' から最後の '}' or ']' までを切り出す方式を採用する。
func cleanJSONResponse(s string) string {
	start := strings.IndexAny(s, "[{")
	end := strings.LastIndexAny(s, "}]")
	if start >= 0 && end >= 0 && start < end {
		return s[start : end+1]
	}
	return strings.TrimSpace(s)
}
