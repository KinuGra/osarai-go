package db

import (
	"context"
	"database/sql"
)

type reviewLogStore struct{ db *sql.DB }

func NewReviewLogStore(db *sql.DB) ReviewLogStore { return &reviewLogStore{db} }

func (s *reviewLogStore) Append(_ context.Context, _ *ReviewLog) error {
	panic("TODO: implement")
}

func (s *reviewLogStore) ListByReview(_ context.Context, _ int64) ([]*ReviewLog, error) {
	panic("TODO: implement")
}
