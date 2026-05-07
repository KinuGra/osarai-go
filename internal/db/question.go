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
		 (session_id, commit_id, title, body, question_type, choices, answer, explanation, saved_for_md)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		question.SessionID, question.CommitID, question.Title, question.Body,
		question.QuestionType, question.Choices, question.Answer, question.Explanation,
		question.SavedForMD,
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
		`SELECT id, session_id, commit_id, title, body, question_type, choices, answer, explanation, saved_for_md, created_at
		 FROM questions WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("question list by session: %w", err)
	}
	defer rows.Close()

	var qs []Question
	for rows.Next() {
		var q Question
		var commitID sql.NullInt64
		var createdAt []byte
		if err := rows.Scan(
			&q.ID, &q.SessionID, &commitID, &q.Title, &q.Body,
			&q.QuestionType, &q.Choices, &q.Answer, &q.Explanation, &q.SavedForMD, &createdAt,
		); err != nil {
			return nil, err
		}
		if commitID.Valid {
			q.CommitID = &commitID.Int64
		}
		if t, err := parseTime(createdAt); err == nil {
			q.CreatedAt = t
		}
		qs = append(qs, q)
	}
	return qs, rows.Err()
}

func (s *questionStore) FindSaved() ([]Question, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, commit_id, title, body, question_type, choices, answer, explanation, saved_for_md, created_at
		 FROM questions WHERE saved_for_md = 1 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("question find saved: %w", err)
	}
	defer rows.Close()

	var qs []Question
	for rows.Next() {
		var q Question
		var commitID sql.NullInt64
		var createdAt []byte
		if err := rows.Scan(
			&q.ID, &q.SessionID, &commitID, &q.Title, &q.Body,
			&q.QuestionType, &q.Choices, &q.Answer, &q.Explanation, &q.SavedForMD, &createdAt,
		); err != nil {
			return nil, err
		}
		if commitID.Valid {
			q.CommitID = &commitID.Int64
		}
		if t, err := parseTime(createdAt); err == nil {
			q.CreatedAt = t
		}
		qs = append(qs, q)
	}
	return qs, rows.Err()
}

func (s *questionStore) MarkSavedForMD(id int64) error {
	_, err := s.db.Exec(`UPDATE questions SET saved_for_md = 1 WHERE id = ?`, id)
	return err
}
