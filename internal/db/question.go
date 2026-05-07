package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type questionStore struct{ db *sql.DB }

func NewQuestionStore(db *sql.DB) QuestionStore { return &questionStore{db} }

func (s *questionStore) Save(ctx context.Context, q *Question) error {
	choices, err := json.Marshal(q.Choices)
	if err != nil {
		return fmt.Errorf("marshal choices: %w", err)
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO questions
		 (session_id, commit_id, title, content, question_type, choices, correct_answer, category, saved)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.SessionID, q.CommitID, q.Title, q.Content, q.QuestionType,
		string(choices), q.CorrectAnswer, q.Category, q.Saved,
	)
	if err != nil {
		return fmt.Errorf("question save: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("question last insert id: %w", err)
	}
	q.ID = id
	return nil
}

func (s *questionStore) FindByID(ctx context.Context, id int64) (*Question, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, commit_id, title, content, question_type, choices, correct_answer, category, saved, created_at
		 FROM questions WHERE id=?`, id,
	)
	q := &Question{}
	var commitID sql.NullInt64
	var choicesJSON string
	var createdAt []byte
	if err := row.Scan(
		&q.ID, &q.SessionID, &commitID, &q.Title, &q.Content,
		&q.QuestionType, &choicesJSON, &q.CorrectAnswer, &q.Category, &q.Saved, &createdAt,
	); err != nil {
		return nil, fmt.Errorf("question find: %w", err)
	}
	if commitID.Valid {
		q.CommitID = &commitID.Int64
	}
	if err := json.Unmarshal([]byte(choicesJSON), &q.Choices); err != nil {
		q.Choices = nil
	}
	if t, err := parseTime(createdAt); err == nil {
		q.CreatedAt = t
	}
	return q, nil
}

func (s *questionStore) ListBySession(ctx context.Context, sessionID int64) ([]*Question, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, commit_id, title, content, question_type, choices, correct_answer, category, saved, created_at
		 FROM questions WHERE session_id=? ORDER BY id`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("question list: %w", err)
	}
	defer rows.Close()

	var qs []*Question
	for rows.Next() {
		q := &Question{}
		var commitID sql.NullInt64
		var choicesJSON string
		var createdAt []byte
		if err := rows.Scan(
			&q.ID, &q.SessionID, &commitID, &q.Title, &q.Content,
			&q.QuestionType, &choicesJSON, &q.CorrectAnswer, &q.Category, &q.Saved, &createdAt,
		); err != nil {
			return nil, err
		}
		if commitID.Valid {
			q.CommitID = &commitID.Int64
		}
		if err := json.Unmarshal([]byte(choicesJSON), &q.Choices); err != nil {
			q.Choices = nil
		}
		if t, err := parseTime(createdAt); err == nil {
			q.CreatedAt = t
		}
		qs = append(qs, q)
	}
	return qs, rows.Err()
}

func (s *questionStore) MarkSaved(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE questions SET saved=1 WHERE id=?`, id)
	return err
}
