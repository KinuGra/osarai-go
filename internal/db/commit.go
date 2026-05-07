package db

import (
	"database/sql"
)

type commitStore struct{ db *sql.DB }

func NewCommitStore(db *sql.DB) CommitStore { return &commitStore{db} }

func (s *commitStore) Create(_ *Commit) error {
	panic("TODO: implement")
}

func (s *commitStore) FindUnreviewed(_ int64) ([]Commit, error) {
	panic("TODO: implement")
}

func (s *commitStore) MarkReviewed(_ int64) error {
	panic("TODO: implement")
}
