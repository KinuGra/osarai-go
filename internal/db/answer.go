package db

import (
	"context"
	"database/sql"
)

type answerStore struct{ db *sql.DB }

func NewAnswerStore(db *sql.DB) AnswerStore { return &answerStore{db} }

func (s *answerStore) Save(_ context.Context, _ *Answer) error {
	panic("TODO: implement")
}

func (s *answerStore) FindByQuestion(_ context.Context, _ int64) (*Answer, error) {
	panic("TODO: implement")
}

func (s *answerStore) ListRetryable(_ context.Context) ([]*Answer, error) {
	panic("TODO: implement")
}
