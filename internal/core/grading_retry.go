package core

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/db"
)

// retryGrading は grade_status="failed_retryable" の回答を AI で再採点する。
// 再採点に成功した件数を返す。
func (s *Service) retryGrading(ctx context.Context) (int, error) {
	if s.grader == nil {
		return 0, fmt.Errorf("grader が設定されていません")
	}

	retryable, err := s.answerStore.FindRetryable()
	if err != nil {
		return 0, fmt.Errorf("再採点対象の取得に失敗: %w", err)
	}
	if len(retryable) == 0 {
		return 0, nil
	}

	successCount := 0
	for _, answer := range retryable {
		q, err := s.questionStore.FindByID(answer.QuestionID)
		if err != nil || q == nil {
			continue
		}

		aiQ := dbQuestionToAIQuestion(q)
		diffCtx := derefStr(q.DiffContext)

		result, err := s.grader.GradeAnswer(ctx, ai.GradeRequest{
			Question:    aiQ,
			UserAnswer:  answer.UserAnswer,
			DiffContext: diffCtx,
		})
		if err != nil || result.GradeStatus == ai.GradeStatusFailedRetryable {
			continue
		}

		// 採点成功: grade_status と explanation を更新
		var expl *string
		if result.Explanation != "" {
			expl = &result.Explanation
		}
		if err := s.answerStore.UpdateGradeStatus(answer.ID, string(result.GradeStatus), expl); err != nil {
			continue
		}

		// reviews レコードを新規作成（初回採点失敗時に作られなかったため）
		review := &db.Review{
			AnswerID:     answer.ID,
			EaseFactor:   2.5,
			IntervalDays: 1,
			Repetitions:  0,
			NextReviewAt: time.Now().AddDate(0, 0, 1),
		}
		if err := s.reviewStore.Create(review); err != nil {
			fmt.Fprintf(os.Stderr, "warning: SM-2 review の作成に失敗: %v\n", err)
		}

		successCount++
	}

	return successCount, nil
}
