package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	_ "embed"
)

//go:embed templates/export.md.tmpl
var exportTmplSrc string

var exportTmpl = template.Must(template.New("export").Parse(exportTmplSrc))

// exportData はテンプレートに渡すデータ構造。
type exportData struct {
	Title         string
	Category      string
	CreatedAt     string
	SelfRating    string
	ScoreLabel    string
	Body          string
	Choices       []string
	UserAnswer    string
	IsCorrect     bool
	CorrectAnswer string
	Explanation   string
}

// runExport は保存済み回答を全て md にエクスポートする。
func (s *Service) runExport(ctx context.Context, opts ExportOptions) error {
	answers, err := s.answerStore.FindSaved()
	if err != nil {
		return fmt.Errorf("保存済み回答の取得に失敗: %w", err)
	}
	if len(answers) == 0 {
		return fmt.Errorf("保存済みの問題がありません。クイズ後に「保存する」を選んでください")
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".osarai", "exports")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("エクスポートディレクトリの作成に失敗: %w", err)
	}

	for _, answer := range answers {
		if _, err := s.exportAnswer(ctx, answer.ID, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: answer %d のエクスポートに失敗: %v\n", answer.ID, err)
		}
	}
	return nil
}

// exportAnswer は 1 件の回答を md ファイルにエクスポートし、パスを返す。
func (s *Service) exportAnswer(_ context.Context, answerID int64, outputDir string) (string, error) {
	answer, err := s.answerStore.FindByID(answerID)
	if err != nil || answer == nil {
		return "", fmt.Errorf("回答 %d の取得に失敗", answerID)
	}
	q, err := s.questionStore.FindByID(answer.QuestionID)
	if err != nil || q == nil {
		return "", fmt.Errorf("問題の取得に失敗")
	}

	var choices []string
	if q.Choices != "" {
		_ = json.Unmarshal([]byte(q.Choices), &choices)
	}

	isCorrect := false
	if answer.IsCorrect != nil {
		isCorrect = *answer.IsCorrect
	}
	scoreLabel := ""
	if answer.AIScoreLabel != nil {
		scoreLabel = *answer.AIScoreLabel
	}
	explanation := ""
	if answer.AIExplanation != nil {
		explanation = *answer.AIExplanation
	}

	title := ensureNonemptyTitle(q.Title, q.ID)
	data := exportData{
		Title:         title,
		Category:      q.Category,
		CreatedAt:     answer.CreatedAt.Format("2006-01-02"),
		SelfRating:    "", // review_log から取得する場合は別途実装
		ScoreLabel:    scoreLabel,
		Body:          q.Body,
		Choices:       choices,
		UserAnswer:    answer.UserAnswer,
		IsCorrect:     isCorrect,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   explanation,
	}

	var buf bytes.Buffer
	if err := exportTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("テンプレートのレンダリングに失敗: %w", err)
	}

	filename := buildExportFilename(answer.CreatedAt, title)
	path := filepath.Join(outputDir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("ファイルの書き込みに失敗: %w", err)
	}

	if err := s.answerStore.MarkSaved(answerID, path); err != nil {
		fmt.Fprintf(os.Stderr, "warning: export_path の更新に失敗: %v\n", err)
	}

	return path, nil
}

// buildExportFilename は "YYYY-MM-DD タイトル.md" 形式のファイル名を生成する。
// OS 禁止文字を _ に置換し、100 文字に切り詰める。
func buildExportFilename(t time.Time, title string) string {
	re := regexp.MustCompile(`[/\\:*?"<>|]`)
	sanitized := re.ReplaceAllString(title, "_")
	sanitized = strings.TrimSpace(sanitized)
	if len([]rune(sanitized)) > 80 {
		runes := []rune(sanitized)
		sanitized = string(runes[:80])
	}
	return fmt.Sprintf("%s %s.md", t.Format("2006-01-02"), sanitized)
}

// listSaved は保存済みアイテムを返す。
func (s *Service) listSaved() ([]SavedItem, error) {
	answers, err := s.answerStore.FindSaved()
	if err != nil {
		return nil, fmt.Errorf("保存済み回答の取得に失敗: %w", err)
	}

	var items []SavedItem
	for _, answer := range answers {
		q, err := s.questionStore.FindByID(answer.QuestionID)
		if err != nil || q == nil {
			continue
		}
		exportPath := ""
		if answer.ExportPath != nil {
			exportPath = *answer.ExportPath
		}
		items = append(items, SavedItem{
			AnswerID:   answer.ID,
			Title:      ensureNonemptyTitle(q.Title, q.ID),
			Category:   q.Category,
			ExportPath: exportPath,
			CreatedAt:  answer.CreatedAt.Format("2006-01-02"),
		})
	}
	return items, nil
}
