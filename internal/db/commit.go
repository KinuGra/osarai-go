package db

import (
	"database/sql"
	"errors"
	"fmt"
)

type commitStore struct{ db *sql.DB }

func NewCommitStore(db *sql.DB) CommitStore { return &commitStore{db} }

func (s *commitStore) Create(commit *Commit) error {
	res, err := s.db.Exec(
		`INSERT INTO commits
		 (repository_id, hash, message, author_name, author_email, diff_summary, reviewed, committed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		commit.RepositoryID, commit.Hash, commit.Message,
		commit.AuthorName, commit.AuthorEmail,
		commit.DiffSummary, commit.Reviewed,
		formatTime(commit.CommittedAt),
	)
	if err != nil {
		return fmt.Errorf("commit create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("commit last insert id: %w", err)
	}
	commit.ID = id
	return nil
}

func (s *commitStore) FindByHash(repoID int64, hash string) (*Commit, error) {
	row := s.db.QueryRow(
		`SELECT id, repository_id, hash, message, author_name, author_email,
		        diff_summary, reviewed, committed_at, created_at
		 FROM commits WHERE repository_id = ? AND hash = ?`,
		repoID, hash,
	)
	c, err := scanCommit(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (s *commitStore) FindUnreviewed(repoID int64) ([]Commit, error) {
	rows, err := s.db.Query(
		`SELECT id, repository_id, hash, message, author_name, author_email,
		        diff_summary, reviewed, committed_at, created_at
		 FROM commits
		 WHERE reviewed = 0 AND repository_id = ?
		 ORDER BY committed_at DESC`,
		repoID,
	)
	if err != nil {
		return nil, fmt.Errorf("commit find unreviewed: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	return scanCommits(rows)
}

func (s *commitStore) MarkReviewed(id int64) error {
	_, err := s.db.Exec(`UPDATE commits SET reviewed = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("commit mark reviewed: %w", err)
	}
	return nil
}

func scanCommit(row *sql.Row) (*Commit, error) {
	var c Commit
	var diffSummary sql.NullString
	var committedAt, createdAt []byte
	if err := row.Scan(
		&c.ID, &c.RepositoryID, &c.Hash, &c.Message, &c.AuthorName, &c.AuthorEmail,
		&diffSummary, &c.Reviewed, &committedAt, &createdAt,
	); err != nil {
		return nil, err
	}
	if diffSummary.Valid {
		c.DiffSummary = &diffSummary.String
	}
	t, err := parseTime(committedAt)
	if err != nil {
		return nil, fmt.Errorf("commit parse committed_at: %w", err)
	}
	c.CommittedAt = t
	t, err = parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("commit parse created_at: %w", err)
	}
	c.CreatedAt = t
	return &c, nil
}

func scanCommits(rows *sql.Rows) ([]Commit, error) {
	var commits []Commit
	for rows.Next() {
		var c Commit
		var diffSummary sql.NullString
		var committedAt, createdAt []byte
		if err := rows.Scan(
			&c.ID, &c.RepositoryID, &c.Hash, &c.Message, &c.AuthorName, &c.AuthorEmail,
			&diffSummary, &c.Reviewed, &committedAt, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("commit scan: %w", err)
		}
		if diffSummary.Valid {
			c.DiffSummary = &diffSummary.String
		}
		t, err := parseTime(committedAt)
		if err != nil {
			return nil, fmt.Errorf("commit parse committed_at: %w", err)
		}
		c.CommittedAt = t
		t, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("commit parse created_at: %w", err)
		}
		c.CreatedAt = t
		commits = append(commits, c)
	}
	return commits, rows.Err()
}
