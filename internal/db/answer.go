package db

import (
	"database/sql"
)

type answerStore struct{ db *sql.DB }

func NewAnswerStore(db *sql.DB) AnswerStore { return &answerStore{db} }

func (s *answerStore) Save(_ *Answer) error {
	panic("TODO: implement")
}

func (s *answerStore) FindByQuestionID(_ int64) (*Answer, error) {
	panic("TODO: implement")
}

func (s *answerStore) FindRetryable() ([]Answer, error) {
	panic("TODO: implement")
}

func (s *answerStore) UpdateGradeStatus(_ int64, _ string) error {
	panic("TODO: implement")
}
