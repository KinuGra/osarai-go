package db

import (
	"database/sql"
	"fmt"
)

type questionStore struct{ db *sql.DB }

func NewQuestionStore(db *sql.DB) QuestionStore { return &questionStore{db} }

func (s *questionStore) Save(question *Question) error {
	res, err := s.db.Exec(
		`INSERT INTO questions
		 (session_id, commit_id, title, category, question_type, body,
		  choices, correct_answer, diff_context, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		question.SessionID, question.CommitID, question.Title, question.Category,
		question.QuestionType, question.Body, question.Choices, question.CorrectAnswer,
		question.DiffContext, question.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("question save: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("question last insert id: %w", err)
	}
	question.ID = id
	return nil
}

func (s *questionStore) FindBySessionID(sessionID int64) ([]Question, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, commit_id, title, category, question_type,
		        body, choices, correct_answer, diff_context, sort_order, created_at
		 FROM questions WHERE session_id = ? ORDER BY sort_order`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("question find by session: %w", err)
	}
	defer rows.Close()

	var qs []Question
	for rows.Next() {
		var q Question
		var commitID sql.NullInt64
		var diffContext sql.NullString
		var createdAt []byte
		if err := rows.Scan(
			&q.ID, &q.SessionID, &commitID, &q.Title, &q.Category, &q.QuestionType,
			&q.Body, &q.Choices, &q.CorrectAnswer, &diffContext, &q.SortOrder, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("question scan: %w", err)
		}
		if commitID.Valid {
			q.CommitID = &commitID.Int64
		}
		if diffContext.Valid {
			q.DiffContext = &diffContext.String
		}
		t, err := parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("question parse created_at: %w", err)
		}
		q.CreatedAt = t
		qs = append(qs, q)
	}
	return qs, rows.Err()
}
