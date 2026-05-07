package db

import (
	"database/sql"
	"time"
)

type reviewStore struct{ db *sql.DB }

func NewReviewStore(db *sql.DB) ReviewStore { return &reviewStore{db} }

func (s *reviewStore) Create(_ *Review) error {
	panic("TODO: implement")
}

func (s *reviewStore) FindDue(_ time.Time) ([]Review, error) {
	panic("TODO: implement")
}

func (s *reviewStore) Update(_ *Review) error {
	panic("TODO: implement")
}
