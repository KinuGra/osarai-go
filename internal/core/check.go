package core

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/db"
	"github.com/KinuGra/osarai-go/internal/git"
)

// CheckQuestion は DB 保存済みの問題 ID と AI 問題データを合わせたもの。
// TUI はこの型を使って出題 → 回答 → 採点 → 自己評価を進める。
type CheckQuestion struct {
	DBQuestionID int64
	DBSessionID  int64
	Question     ai.Question
	DiffContext  string // diff の断片（採点時の文脈用）
}

// runCheck は未コミット差分から問題を生成して DB に保存し、[]CheckQuestion を返す。
// Service.RunCheck から呼ばれる。
func (s *Service) runCheck(ctx context.Context, opts CheckOptions) ([]CheckQuestion, error) {
	// 1. リポジトリを検出
	repo, err := git.DetectRepo()
	if err != nil {
		return nil, err
	}

	// 2. DB にリポジトリを登録（パスで既存チェック → なければ作成）
	dbRepo, err := s.repoStore.FindByPath(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("リポジトリの検索に失敗: %w", err)
	}
	if dbRepo == nil {
		dbRepo = &db.Repository{
			Name: repo.Name,
			Path: repo.Path,
		}
		if err := s.repoStore.Create(dbRepo); err != nil {
			return nil, fmt.Errorf("リポジトリの登録に失敗: %w", err)
		}
	}

	// 3. diff を取得
	scope := git.DiffAll
	if opts.Staged {
		scope = git.DiffStaged
	}
	diff, err := git.GetDiff(repo, scope, "")
	if err != nil {
		return nil, err
	}

	// 4. セッションを作成
	sourceRef := "all"
	if opts.Staged {
		sourceRef = "staged"
	}
	if opts.FilePath != "" {
		sourceRef = opts.FilePath
	}

	session := &db.Session{
		Mode:         "check",
		RepositoryID: &dbRepo.ID,
		SourceRef:    &sourceRef,
		StartedAt:    time.Now(),
	}
	if err := s.sessionStore.Create(session); err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	// 5. AI で問題を生成
	aiQuestions, err := s.generator.GenerateQuestions(ctx, ai.GenerateRequest{
		Diff:     diff.Body,
		Language: detectLanguage(diff.Body),
		FilePath: opts.FilePath,
	})
	if err != nil {
		return nil, err
	}

	// 6. 問題を DB に保存し、CheckQuestion リストを構築
	diffCtx := truncateDiff(diff.Body, 2000)
	result := make([]CheckQuestion, 0, len(aiQuestions))

	for i, q := range aiQuestions {
		choicesJSON := "[]"
		if len(q.Choices) > 0 {
			b, err := json.Marshal(q.Choices)
			if err != nil {
				return nil, fmt.Errorf("選択肢の JSON 変換に失敗: %w", err)
			}
			choicesJSON = string(b)
		}

		var explanationPtr *string
		if q.Explanation != "" {
			explanationPtr = &q.Explanation
		}
		dbQ := &db.Question{
			SessionID:     session.ID,
			CommitID:      nil, // 未コミット差分なので NULL
			Title:         q.Title,
			Category:      string(q.Category),
			QuestionType:  string(q.Type),
			Body:          q.Body,
			Choices:       choicesJSON,
			CorrectAnswer: q.CorrectAnswer,
			Explanation:   explanationPtr,
			DiffContext:   &diffCtx,
			SortOrder:     i,
		}

		if err := s.questionStore.Save(dbQ); err != nil {
			return nil, fmt.Errorf("問題の保存に失敗: %w", err)
		}

		result = append(result, CheckQuestion{
			DBQuestionID: dbQ.ID,
			DBSessionID:  session.ID,
			Question:     q,
			DiffContext:  diffCtx,
		})
	}

	return result, nil
}

// detectLanguage は diff テキストの "diff --git" 行からファイル拡張子を集計し、
// 最も多く登場する言語名を返す。判定できない場合は "Unknown" を返す。
func detectLanguage(diffBody string) string {
	extToLang := map[string]string{
		".go":   "Go",
		".ts":   "TypeScript",
		".tsx":  "TypeScript",
		".js":   "JavaScript",
		".jsx":  "JavaScript",
		".py":   "Python",
		".rs":   "Rust",
		".java": "Java",
		".rb":   "Ruby",
		".kt":   "Kotlin",
		".swift": "Swift",
		".cs":   "C#",
		".cpp":  "C++",
		".c":    "C",
		".md":   "Markdown",
	}

	counts := map[string]int{}
	for _, line := range strings.Split(diffBody, "\n") {
		if !strings.HasPrefix(line, "diff --git ") {
			continue
		}
		// "diff --git a/path/to/file.go b/path/to/file.go" → "a/path/to/file.go"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		ext := filepath.Ext(fields[2])
		if lang, ok := extToLang[ext]; ok {
			counts[lang]++
		}
	}

	best, bestN := "Unknown", 0
	for lang, n := range counts {
		if n > bestN {
			best, bestN = lang, n
		}
	}
	return best
}

// truncateDiff は diff テキストを maxRunes 文字に切り詰める。
// []rune 変換によるメモリ確保を避け、rune 境界のバイトオフセットを直接求める。
func truncateDiff(s string, maxRunes int) string {
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}
