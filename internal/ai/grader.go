package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	_ "embed"
)

// GradeStatus は採点結果のステータス。
type GradeStatus string

const (
	GradeStatusGraded          GradeStatus = "graded"           // AI 採点成功
	GradeStatusLocalOnly       GradeStatus = "local_only"       // 選択式のローカル比較のみ
	GradeStatusFailedRetryable GradeStatus = "failed_retryable" // 記述式 AI 採点失敗（retry-grading 対象）
)

// GradeRequest は採点リクエスト。
type GradeRequest struct {
	Question    Question // 元の問題
	UserAnswer  string   // ユーザーの回答
	DiffContext string   // 元の diff（文脈として AI に渡す）
}

// GradeResult は AI による採点結果。
// json タグは LLM レスポンスの JSON キーと一致させる必要がある。
type GradeResult struct {
	IsCorrect   *bool       `json:"is_correct"`  // nil = 採点不能
	Score       *float64    `json:"score"`       // 0.0〜10.0。nil = 採点不能
	ScoreLabel  string      `json:"score_label"` // 表示用（例: "9/10"）
	Explanation string      `json:"explanation"` // 参考書テキスト風の解説
	GradeStatus GradeStatus `json:"grade_status"`
}

// Grader はユーザーの回答を採点する構造体。
type Grader struct {
	provider LLMProvider
}

// NewGrader は Grader を生成する。
func NewGrader(provider LLMProvider) *Grader {
	return &Grader{provider: provider}
}

//go:embed prompts/grade_answer.tmpl
var gradeAnswerTmplSrc string

var gradeAnswerTmpl = template.Must(
	template.New("grade_answer").Parse(gradeAnswerTmplSrc),
)

const gradeSystemPrompt = "あなたはソフトウェアエンジニアの教育専門家です。\n" +
	"記述式問題のユーザー回答を採点し、結果を JSON で返します。\n" +
	"返答は純粋な JSON オブジェクトのみにしてください。コードブロック（```）、説明文、前置き、後書きは一切不要です。\n" +
	"最初の文字は \"{\" でなければなりません。"

// GradeAnswer はユーザーの回答を採点し、参考書風の解説を生成する。
//
// 採点ロジック:
//   - 選択式: ローカル比較（API 不要、コスト節約）。q.Explanation があれば解説に使用。
//   - 記述式: Gemini API で採点。失敗時は GradeStatusFailedRetryable を返す（error は nil）。
func (g *Grader) GradeAnswer(ctx context.Context, req GradeRequest) (GradeResult, error) {
	if req.Question.Type == QuestionTypeChoice {
		return g.gradeChoice(req), nil
	}
	return g.gradeWritten(ctx, req)
}

// gradeChoice は選択式をローカル比較で採点する。
func (g *Grader) gradeChoice(req GradeRequest) GradeResult {
	isCorrect := req.Question.CorrectAnswer == req.UserAnswer
	explanation := buildChoiceExplanation(req.Question, isCorrect)
	return GradeResult{
		IsCorrect:   &isCorrect,
		Explanation: explanation,
		GradeStatus: GradeStatusLocalOnly,
	}
}

// buildChoiceExplanation は選択式の解説文を組み立てる。
// q.Explanation（問題生成時に AI が出力した解説）があればそれを採用し、
// なければ正解ラベルのみ表示する。
func buildChoiceExplanation(q Question, isCorrect bool) string {
	if q.Explanation != "" {
		return fmt.Sprintf("正解は **%s** です。\n\n%s", q.CorrectAnswer, q.Explanation)
	}
	if isCorrect {
		return fmt.Sprintf("正解は **%s** です。", q.CorrectAnswer)
	}
	return fmt.Sprintf("不正解です。正解は **%s** です。", q.CorrectAnswer)
}

// gradeWritten は記述式を Gemini API で採点する。
// API 呼び出し失敗時は error を返さず GradeStatusFailedRetryable を返す。
// これにより呼び出し側は「採点失敗」として扱い、retry-grading で後から再試行できる。
func (g *Grader) gradeWritten(ctx context.Context, req GradeRequest) (GradeResult, error) {
	var buf bytes.Buffer
	if err := gradeAnswerTmpl.Execute(&buf, req); err != nil {
		// テンプレートエラーはプログラムバグなのでエラーとして伝播
		return GradeResult{}, fmt.Errorf("採点テンプレートの適用に失敗: %w", err)
	}

	resp, err := g.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: gradeSystemPrompt,
		UserPrompt:   buf.String(),
		Temperature:  0.3, // 採点は低温度で決定論的に
		MaxTokens:    1024,
	})
	if err != nil {
		// API エラー: failed_retryable として返す（error は nil）
		return GradeResult{GradeStatus: GradeStatusFailedRetryable}, nil
	}

	content := cleanJSONResponse(resp.Content)

	var result GradeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// JSON パース失敗: failed_retryable として返す
		return GradeResult{GradeStatus: GradeStatusFailedRetryable}, nil
	}

	result.GradeStatus = GradeStatusGraded
	return result, nil
}
