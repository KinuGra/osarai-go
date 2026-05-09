package db

import (
	"database/sql"
	"errors"
	"fmt"
)

type repositoryStore struct{ db *sql.DB }

func NewRepositoryStore(db *sql.DB) RepositoryStore { return &repositoryStore{db} }

func (s *repositoryStore) FindByPath(path string) (*Repository, error) {
	row := s.db.QueryRow(
		`SELECT id, name, path, remote_url, created_at
		 FROM repositories WHERE path = ?`, path,
	)
	var r Repository
	var remoteURL sql.NullString
	var createdAt []byte

	err := row.Scan(&r.ID, &r.Name, &r.Path, &remoteURL, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 見つからない場合は nil, nil
		}
		return nil, fmt.Errorf("repository find by path: %w", err)
	}

	if remoteURL.Valid {
		r.RemoteURL = &remoteURL.String
	}
	t, err := parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("repository parse created_at: %w", err)
	}
	r.CreatedAt = t
	return &r, nil
}

func (s *repositoryStore) Create(repo *Repository) error {
	res, err := s.db.Exec(
		`INSERT INTO repositories (name, path, remote_url) VALUES (?, ?, ?)`,
		repo.Name, repo.Path, repo.RemoteURL,
	)
	if err != nil {
		return fmt.Errorf("repository create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("repository last insert id: %w", err)
	}
	repo.ID = id
	return nil
}
