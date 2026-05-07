package db

import (
	"database/sql"
)

type reviewLogStore struct{ db *sql.DB }

func NewReviewLogStore(db *sql.DB) ReviewLogStore { return &reviewLogStore{db} }

func (s *reviewLogStore) Create(_ *ReviewLog) error {
	panic("TODO: implement")
}

func (s *reviewLogStore) FindByReviewID(_ int64) ([]ReviewLog, error) {
	panic("TODO: implement")
}
