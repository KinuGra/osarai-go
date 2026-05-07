package db

import (
	"database/sql"
	"fmt"
)

type sessionStore struct{ db *sql.DB }

func NewSessionStore(db *sql.DB) SessionStore { return &sessionStore{db} }

func (s *sessionStore) Create(session *Session) error {
	res, err := s.db.Exec(
		`INSERT INTO sessions (repository_id, commit_hash, diff_scope, started_at)
		 VALUES (?, ?, ?, ?)`,
		session.RepositoryID, session.CommitHash, session.DiffScope, session.StartedAt,
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

func (s *sessionStore) Finish(id int64) error {
	_, err := s.db.Exec(
		`UPDATE sessions SET finished_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("session finish: %w", err)
	}
	return nil
}
