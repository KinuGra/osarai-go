package core

import (
	"context"
	"fmt"
	"time"
)

// getStats は学習統計を集計して返す。
func (s *Service) getStats(_ context.Context) (Stats, error) {
	retryable, err := s.answerStore.FindRetryable()
	if err != nil {
		return Stats{}, fmt.Errorf("再採点対象の取得に失敗: %w", err)
	}

	saved, err := s.answerStore.FindSaved()
	if err != nil {
		return Stats{}, fmt.Errorf("保存済み回答の取得に失敗: %w", err)
	}

	dueReviews, err := s.reviewStore.FindDue(time.Now())
	if err != nil {
		return Stats{}, fmt.Errorf("復習対象の取得に失敗: %w", err)
	}

	// saved 回答から正答率を算出
	correctAnswers := 0
	for _, a := range saved {
		if a.IsCorrect != nil && *a.IsCorrect {
			correctAnswers++
		}
	}

	return Stats{
		TotalQuestions: len(saved) + len(retryable),
		CorrectAnswers: correctAnswers,
		DueReviews:     len(dueReviews),
		RetryableCount: len(retryable),
	}, nil
}
