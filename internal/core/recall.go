package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/db"
	"github.com/KinuGra/osarai-go/internal/git"
)

// runRecall は SM-2 復習対象 + 未振り返りコミットから問題を取得して返す。
// Service.RunRecall から呼ばれる。
func (s *Service) runRecall(ctx context.Context, opts RecallOptions) ([]CheckQuestion, error) {
	// 1. リポジトリを検出
	repo, err := git.DetectRepo()
	if err != nil {
		return nil, err
	}

	// 2. DB にリポジトリを登録
	dbRepo, err := s.repoStore.FindByPath(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("リポジトリの検索に失敗: %w", err)
	}
	if dbRepo == nil {
		dbRepo = &db.Repository{Name: repo.Name, Path: repo.Path}
		if err := s.repoStore.Create(dbRepo); err != nil {
			return nil, fmt.Errorf("リポジトリの登録に失敗: %w", err)
		}
	}

	// 3. recall セッションを作成
	session := &db.Session{
		Mode:         "recall",
		RepositoryID: &dbRepo.ID,
		StartedAt:    time.Now(),
	}
	if err := s.sessionStore.Create(session); err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	var questions []CheckQuestion

	// 4. SM-2 復習対象を取得（next_review_at <= today）
	dueReviews, err := s.reviewStore.FindDue(time.Now())
	if err != nil {
		return nil, fmt.Errorf("復習対象の取得に失敗: %w", err)
	}
	for _, review := range dueReviews {
		// 回答を取得
		answer, err := s.answerStore.FindByID(review.AnswerID)
		if err != nil || answer == nil {
			continue
		}
		// 問題を取得
		q, err := s.questionStore.FindByID(answer.QuestionID)
		if err != nil || q == nil {
			continue
		}
		aiQ := dbQuestionToAIQuestion(q)
		existingResult := buildExistingGradeResult(answer, q)
		questions = append(questions, CheckQuestion{
			DBQuestionID:     q.ID,
			DBSessionID:      session.ID,
			Question:         aiQ,
			DiffContext:      derefStr(q.DiffContext),
			IsRecallReview:   true,
			ExistingAnswerID: answer.ID,
			ExistingResult:   &existingResult,
		})
	}

	// 5. git log から未登録コミットを検出して DB に追加
	authorEmail, _ := git.GetAuthorEmail(repo)
	gitLog, err := git.GetCommitLog(repo, authorEmail, 30)
	if err == nil {
		for _, ci := range gitLog {
			existing, _ := s.commitStore.FindByHash(dbRepo.ID, ci.Hash)
			if existing != nil {
				continue
			}
			commit := &db.Commit{
				RepositoryID: dbRepo.ID,
				Hash:         ci.Hash,
				Message:      ci.Message,
				AuthorName:   ci.AuthorName,
				AuthorEmail:  ci.AuthorEmail,
				CommittedAt:  ci.CommittedAt,
			}
			_ = s.commitStore.Create(commit)
		}
	}

	// 6. 未振り返りコミットから問題を生成
	unreviewed, err := s.commitStore.FindUnreviewed(dbRepo.ID)
	if err != nil {
		return nil, fmt.Errorf("未振り返りコミットの取得に失敗: %w", err)
	}
	for _, commit := range unreviewed {
		diff, err := git.GetDiff(repo, git.DiffCommit, commit.Hash)
		if err != nil {
			continue
		}
		aiQuestions, err := s.generator.GenerateQuestions(ctx, ai.GenerateRequest{
			Diff:     diff.Body,
			Language: detectLanguage(diff.Body),
		})
		if err != nil {
			continue
		}
		diffCtx := truncateDiff(diff.Body, 2000)
		for i, q := range aiQuestions {
			choicesJSON := "[]"
			if len(q.Choices) > 0 {
				b, _ := json.Marshal(q.Choices)
				choicesJSON = string(b)
			}
			var explanationPtr *string
			if q.Explanation != "" {
				explanationPtr = &q.Explanation
			}
			commitIDVal := commit.ID
			dbQ := &db.Question{
				SessionID:     session.ID,
				CommitID:      &commitIDVal,
				Title:         q.Title,
				Category:      string(q.Category),
				QuestionType:  string(q.Type),
				Body:          q.Body,
				Choices:       choicesJSON,
				CorrectAnswer: q.CorrectAnswer,
				Explanation:   explanationPtr,
				DiffContext:   &diffCtx,
				SortOrder:     len(questions) + i,
			}
			if err := s.questionStore.Save(dbQ); err != nil {
				continue
			}
			questions = append(questions, CheckQuestion{
				DBQuestionID: dbQ.ID,
				DBSessionID:  session.ID,
				Question:     q,
				DiffContext:  diffCtx,
			})
		}
	}

	return questions, nil
}

// dbQuestionToAIQuestion は DB の Question を ai.Question に変換する。
func dbQuestionToAIQuestion(q *db.Question) ai.Question {
	var choices []string
	if q.Choices != "" && q.Choices != "[]" {
		_ = json.Unmarshal([]byte(q.Choices), &choices)
	}
	explanation := ""
	if q.Explanation != nil {
		explanation = *q.Explanation
	}
	return ai.Question{
		Title:         q.Title,
		Category:      ai.QuestionCategory(q.Category),
		Type:          ai.QuestionType(q.QuestionType),
		Body:          q.Body,
		Choices:       choices,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   explanation,
	}
}

// buildExistingGradeResult は既存の DB Answer から GradeAnswerResult を構築する。
func buildExistingGradeResult(answer *db.Answer, q *db.Question) GradeAnswerResult {
	explanation := ""
	if answer.AIExplanation != nil {
		explanation = *answer.AIExplanation
	} else if q.Explanation != nil {
		// AI 解説がなければ問題の解説を使う
		explanation = fmt.Sprintf("正解は **%s** です。\n\n%s", q.CorrectAnswer, *q.Explanation)
	} else {
		explanation = fmt.Sprintf("正解は **%s** です。", q.CorrectAnswer)
	}

	result := ai.GradeResult{
		Explanation: explanation,
		GradeStatus: ai.GradeStatus(answer.GradeStatus),
	}
	if answer.IsCorrect != nil {
		result.IsCorrect = answer.IsCorrect
	}
	if answer.AIScore != nil {
		result.Score = answer.AIScore
	}
	if answer.AIScoreLabel != nil {
		result.ScoreLabel = *answer.AIScoreLabel
	}

	return GradeAnswerResult{
		DBAnswerID: answer.ID,
		Result:     result,
	}
}

// derefStr はポインタ文字列を実体化する。nil なら空文字を返す。
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SaveRatingRequest は SM-2 自己評価の保存リクエスト。
type SaveRatingRequest struct {
	DBAnswerID int64
	Rating     string // "again" | "hard" | "good" | "easy"
}

// saveRating は自己評価を SM-2 に反映して review_logs に追記する。
// Service.SaveRating から呼ばれる。
func (s *Service) saveRating(_ context.Context, req SaveRatingRequest) error {
	review, err := s.reviewStore.FindByAnswerID(req.DBAnswerID)
	if err != nil || review == nil {
		// review が存在しない場合（failed_retryable など）はスキップ
		return nil
	}

	before := SM2Params{
		EaseFactor:  review.EaseFactor,
		Interval:    review.IntervalDays,
		Repetitions: review.Repetitions,
	}
	result := Calculate(before, req.Rating)

	log := &db.ReviewLog{
		AnswerID:         req.DBAnswerID,
		SelfRating:       req.Rating,
		EaseFactorBefore: before.EaseFactor,
		EaseFactorAfter:  result.EaseFactor,
		IntervalBefore:   before.Interval,
		IntervalAfter:    result.Interval,
	}
	if err := s.reviewLogStore.Create(log); err != nil {
		return fmt.Errorf("review log の保存に失敗: %w", err)
	}

	review.EaseFactor = result.EaseFactor
	review.IntervalDays = result.Interval
	review.Repetitions = result.Repetitions
	review.NextReviewAt = result.NextReviewAt
	if err := s.reviewStore.Update(review); err != nil {
		return fmt.Errorf("review の更新に失敗: %w", err)
	}

	return nil
}

// CountRetryable は grade_status='failed_retryable' の件数を返す。
func (s *Service) CountRetryable() (int, error) {
	answers, err := s.answerStore.FindRetryable()
	if err != nil {
		return 0, err
	}
	return len(answers), nil
}

// ensureNonemptyTitle は title が空のときフォールバック文字列を返す。
func ensureNonemptyTitle(title string, questionID int64) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	return fmt.Sprintf("untitled-%d", questionID)
}
