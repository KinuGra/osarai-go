package db

import (
	"database/sql"
	"fmt"
)

type answerStore struct{ db *sql.DB }

func NewAnswerStore(db *sql.DB) AnswerStore { return &answerStore{db} }

func (s *answerStore) Save(answer *Answer) error {
	res, err := s.db.Exec(
		`INSERT INTO answers
		 (question_id, user_answer, is_correct, ai_score, ai_score_label,
		  ai_explanation, grade_status, saved, export_path)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		answer.QuestionID, answer.UserAnswer, boolToNullInt(answer.IsCorrect),
		answer.AIScore, answer.AIScoreLabel, answer.AIExplanation,
		answer.GradeStatus, answer.Saved, answer.ExportPath,
	)
	if err != nil {
		return fmt.Errorf("answer save: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("answer last insert id: %w", err)
	}
	answer.ID = id
	return nil
}

func (s *answerStore) FindByQuestionID(questionID int64) (*Answer, error) {
	row := s.db.QueryRow(
		`SELECT id, question_id, user_answer, is_correct, ai_score, ai_score_label,
		        ai_explanation, grade_status, saved, export_path, created_at
		 FROM answers WHERE question_id = ?`, questionID,
	)
	return scanAnswer(row)
}

func (s *answerStore) FindRetryable() ([]Answer, error) {
	rows, err := s.db.Query(
		`SELECT id, question_id, user_answer, is_correct, ai_score, ai_score_label,
		        ai_explanation, grade_status, saved, export_path, created_at
		 FROM answers WHERE grade_status = 'failed_retryable'`,
	)
	if err != nil {
		return nil, fmt.Errorf("answer find retryable: %w", err)
	}
	defer rows.Close()
	return scanAnswers(rows)
}

func (s *answerStore) UpdateGradeStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE answers SET grade_status = ? WHERE id = ?`, status, id)
	return err
}

func (s *answerStore) MarkSaved(id int64) error {
	_, err := s.db.Exec(`UPDATE answers SET saved = 1 WHERE id = ?`, id)
	return err
}

func (s *answerStore) FindSaved() ([]Answer, error) {
	rows, err := s.db.Query(
		`SELECT id, question_id, user_answer, is_correct, ai_score, ai_score_label,
		        ai_explanation, grade_status, saved, export_path, created_at
		 FROM answers WHERE saved = 1 ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("answer find saved: %w", err)
	}
	defer rows.Close()
	return scanAnswers(rows)
}

func scanAnswer(row *sql.Row) (*Answer, error) {
	var a Answer
	var isCorrect sql.NullInt64
	var aiScore sql.NullFloat64
	var aiScoreLabel, aiExplanation, exportPath sql.NullString
	var createdAt []byte
	if err := row.Scan(
		&a.ID, &a.QuestionID, &a.UserAnswer, &isCorrect, &aiScore,
		&aiScoreLabel, &aiExplanation, &a.GradeStatus, &a.Saved, &exportPath, &createdAt,
	); err != nil {
		return nil, fmt.Errorf("answer scan: %w", err)
	}
	if isCorrect.Valid {
		b := isCorrect.Int64 != 0
		a.IsCorrect = &b
	}
	if aiScore.Valid {
		a.AIScore = &aiScore.Float64
	}
	if aiScoreLabel.Valid {
		a.AIScoreLabel = &aiScoreLabel.String
	}
	if aiExplanation.Valid {
		a.AIExplanation = &aiExplanation.String
	}
	if exportPath.Valid {
		a.ExportPath = &exportPath.String
	}
	t, err := parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("answer parse created_at: %w", err)
	}
	a.CreatedAt = t
	return &a, nil
}

func scanAnswers(rows *sql.Rows) ([]Answer, error) {
	var answers []Answer
	for rows.Next() {
		var a Answer
		var isCorrect sql.NullInt64
		var aiScore sql.NullFloat64
		var aiScoreLabel, aiExplanation, exportPath sql.NullString
		var createdAt []byte
		if err := rows.Scan(
			&a.ID, &a.QuestionID, &a.UserAnswer, &isCorrect, &aiScore,
			&aiScoreLabel, &aiExplanation, &a.GradeStatus, &a.Saved, &exportPath, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("answer scan: %w", err)
		}
		if isCorrect.Valid {
			b := isCorrect.Int64 != 0
			a.IsCorrect = &b
		}
		if aiScore.Valid {
			a.AIScore = &aiScore.Float64
		}
		if aiScoreLabel.Valid {
			a.AIScoreLabel = &aiScoreLabel.String
		}
		if aiExplanation.Valid {
			a.AIExplanation = &aiExplanation.String
		}
		if exportPath.Valid {
			a.ExportPath = &exportPath.String
		}
		t, err := parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("answer parse created_at: %w", err)
		}
		a.CreatedAt = t
		answers = append(answers, a)
	}
	return answers, rows.Err()
}

func boolToNullInt(b *bool) interface{} {
	if b == nil {
		return nil
	}
	if *b {
		return 1
	}
	return 0
}
