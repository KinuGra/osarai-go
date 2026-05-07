package db

import (
	"context"
	"database/sql"
)

type reviewStore struct{ db *sql.DB }

func NewReviewStore(db *sql.DB) ReviewStore { return &reviewStore{db} }

func (s *reviewStore) Create(_ context.Context, _ *Review) error {
	panic("TODO: implement")
}

func (s *reviewStore) Update(_ context.Context, _ *Review) error {
	panic("TODO: implement")
}

func (s *reviewStore) ListDue(_ context.Context) ([]*Review, error) {
	panic("TODO: implement")
}

func (s *reviewStore) FindByQuestion(_ context.Context, _ int64) (*Review, error) {
	panic("TODO: implement")
}
