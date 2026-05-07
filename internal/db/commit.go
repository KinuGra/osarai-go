package db

import (
	"context"
	"database/sql"
)

type commitStore struct{ db *sql.DB }

func NewCommitStore(db *sql.DB) CommitStore { return &commitStore{db} }

func (s *commitStore) Save(_ context.Context, _ *Commit) error {
	panic("TODO: implement")
}

func (s *commitStore) FindByHash(_ context.Context, _ int64, _ string) (*Commit, error) {
	panic("TODO: implement")
}

func (s *commitStore) ListUnreviewed(_ context.Context, _ int64) ([]*Commit, error) {
	panic("TODO: implement")
}

func (s *commitStore) MarkReviewed(_ context.Context, _ int64) error {
	panic("TODO: implement")
}
