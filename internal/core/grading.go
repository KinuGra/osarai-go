package core

import (
	"context"
	"fmt"
	"time"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/db"
)

// GradeAnswerRequest は TUI → core への採点リクエスト。
type GradeAnswerRequest struct {
	DBQuestionID int64
	Question     ai.Question
	UserAnswer   string
	DiffContext  string
}

// GradeAnswerResult は採点結果と DB に保存された Answer の ID。
type GradeAnswerResult struct {
	DBAnswerID int64
	Result     ai.GradeResult
}

// gradeAnswer はユーザーの回答を採点して DB に保存し、結果を返す。
// Service.GradeAnswer から呼ばれる。
//
// MVP フェーズ: grader が注入されていない場合はローカル採点（モック）を使用。
// B（採点実装）が完了した後、実際の ai.Grader に差し替える。
func (s *Service) gradeAnswer(ctx context.Context, req GradeAnswerRequest) (GradeAnswerResult, error) {
	var result ai.GradeResult

	if s.grader != nil {
		// 実採点（ai.Grader が DI されている場合）
		var err error
		result, err = s.grader.GradeAnswer(ctx, ai.GradeRequest{
			Question:    req.Question,
			UserAnswer:  req.UserAnswer,
			DiffContext: req.DiffContext,
		})
		if err != nil {
			// フォールバック: ローカル採点
			result = gradeLocally(req.Question, req.UserAnswer)
		}
	} else {
		// モック採点（grader 未注入の MVP フェーズ）
		result = gradeLocally(req.Question, req.UserAnswer)
	}

	// DB に回答を保存
	answer := buildAnswer(req.DBQuestionID, req.UserAnswer, result)
	if err := s.answerStore.Save(answer); err != nil {
		return GradeAnswerResult{}, fmt.Errorf("回答の保存に失敗: %w", err)
	}

	// SM-2 初期 Review レコードを作成（grade_status が graded または local_only の場合）
	if result.GradeStatus != ai.GradeStatusFailedRetryable {
		review := &db.Review{
			AnswerID:     answer.ID,
			EaseFactor:   2.5,
			IntervalDays: 1,
			Repetitions:  0,
			NextReviewAt: time.Now().AddDate(0, 0, 1), // 翌日復習
		}
		if err := s.reviewStore.Create(review); err != nil {
			// Review 作成失敗は致命的でないのでログのみ（将来的にはログ出力）
			_ = err
		}
	}

	return GradeAnswerResult{
		DBAnswerID: answer.ID,
		Result:     result,
	}, nil
}

// gradeLocally は AI を使わずローカルで採点する（モック / フォールバック）。
func gradeLocally(q ai.Question, userAnswer string) ai.GradeResult {
	if q.Type == ai.QuestionTypeChoice {
		isCorrect := q.CorrectAnswer == userAnswer
		explanation := fmt.Sprintf(
			"正解は **%s** です。\n\n模範解答: %s",
			q.CorrectAnswer, q.CorrectAnswer,
		)
		return ai.GradeResult{
			IsCorrect:   &isCorrect,
			Explanation: explanation,
			GradeStatus: ai.GradeStatusLocalOnly,
		}
	}

	// 記述式: MVP では常に「採点中」として扱い、模範解答を表示
	isCorrect := true
	explanation := fmt.Sprintf("模範解答:\n\n%s", q.CorrectAnswer)
	return ai.GradeResult{
		IsCorrect:   &isCorrect,
		Explanation: explanation,
		GradeStatus: ai.GradeStatusLocalOnly,
	}
}

// buildAnswer は GradeResult から db.Answer を構築する。
func buildAnswer(questionID int64, userAnswer string, r ai.GradeResult) *db.Answer {
	a := &db.Answer{
		QuestionID:  questionID,
		UserAnswer:  userAnswer,
		IsCorrect:   r.IsCorrect,
		GradeStatus: string(r.GradeStatus),
		Saved:       false,
		CreatedAt:   time.Now(),
	}
	if r.Score != nil {
		a.AIScore = r.Score
	}
	if r.ScoreLabel != "" {
		a.AIScoreLabel = &r.ScoreLabel
	}
	if r.Explanation != "" {
		a.AIExplanation = &r.Explanation
	}
	return a
}
