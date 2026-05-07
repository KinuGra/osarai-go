package db

import (
	"database/sql"
)

type repositoryStore struct{ db *sql.DB }

func NewRepositoryStore(db *sql.DB) RepositoryStore { return &repositoryStore{db} }

func (s *repositoryStore) FindByPath(_ string) (*Repository, error) {
	panic("TODO: implement")
}

func (s *repositoryStore) Create(_ *Repository) error {
	panic("TODO: implement")
}
