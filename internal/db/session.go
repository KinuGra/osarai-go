package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type sessionStore struct{ db *sql.DB }

func NewSessionStore(db *sql.DB) SessionStore { return &sessionStore{db} }

func (s *sessionStore) Create(ctx context.Context, sess *Session) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (repository_id, started_at, total_questions, correct_count, max_streak)
		 VALUES (?, ?, ?, ?, ?)`,
		sess.RepositoryID, sess.StartedAt, sess.TotalQuestions, sess.CorrectCount, sess.MaxStreak,
	)
	if err != nil {
		return fmt.Errorf("session create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("session last insert id: %w", err)
	}
	sess.ID = id
	return nil
}

func (s *sessionStore) Update(ctx context.Context, sess *Session) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions
		 SET ended_at=?, total_questions=?, correct_count=?, max_streak=?
		 WHERE id=?`,
		sess.EndedAt, sess.TotalQuestions, sess.CorrectCount, sess.MaxStreak, sess.ID,
	)
	return err
}

func (s *sessionStore) FindByID(ctx context.Context, id int64) (*Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, repository_id, started_at, ended_at, total_questions, correct_count, max_streak
		 FROM sessions WHERE id=?`, id,
	)
	sess := &Session{}
	var startedAt, endedAt []byte
	var repoID sql.NullInt64
	if err := row.Scan(&sess.ID, &repoID, &startedAt, &endedAt, &sess.TotalQuestions, &sess.CorrectCount, &sess.MaxStreak); err != nil {
		return nil, fmt.Errorf("session find: %w", err)
	}
	if repoID.Valid {
		sess.RepositoryID = &repoID.Int64
	}
	if t, err := parseTime(startedAt); err == nil {
		sess.StartedAt = t
	}
	if len(endedAt) > 0 {
		if t, err := parseTime(endedAt); err == nil {
			sess.EndedAt = &t
		}
	}
	return sess, nil
}

func parseTime(b []byte) (time.Time, error) {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		return time.Parse(time.RFC3339, s)
	}
	return time.Parse("2006-01-02 15:04:05", string(b))
}
