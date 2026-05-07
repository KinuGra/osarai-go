package db

import (
	"context"
	"database/sql"
)

type repositoryStore struct{ db *sql.DB }

func NewRepositoryStore(db *sql.DB) RepositoryStore { return &repositoryStore{db} }

func (s *repositoryStore) Save(_ context.Context, _ *Repository) error {
	panic("TODO: implement")
}

func (s *repositoryStore) FindByPath(_ context.Context, _ string) (*Repository, error) {
	panic("TODO: implement")
}

func (s *repositoryStore) List(_ context.Context) ([]*Repository, error) {
	panic("TODO: implement")
}
