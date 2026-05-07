package db

import (
	"database/sql"
	"fmt"
)

type sessionStore struct{ db *sql.DB }

func NewSessionStore(db *sql.DB) SessionStore { return &sessionStore{db} }

func (s *sessionStore) Create(session *Session) error {
	res, err := s.db.Exec(
		`INSERT INTO sessions (mode, repository_id, source_ref, started_at)
		 VALUES (?, ?, ?, ?)`,
		session.Mode, session.RepositoryID, session.SourceRef, formatTime(session.StartedAt),
	)
	if err != nil {
		return fmt.Errorf("session create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("session last insert id: %w", err)
	}
	session.ID = id
	return nil
}

func (s *sessionStore) Finish(id int64, totalQuestions, correctCount, maxStreak int) error {
	_, err := s.db.Exec(
		`UPDATE sessions
		 SET finished_at     = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		     total_questions  = ?,
		     correct_count    = ?,
		     max_streak       = ?
		 WHERE id = ?`,
		totalQuestions, correctCount, maxStreak, id,
	)
	if err != nil {
		return fmt.Errorf("session finish: %w", err)
	}
	return nil
}
