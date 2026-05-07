package db

import (
	"database/sql"
	"fmt"
)

type reviewLogStore struct{ db *sql.DB }

func NewReviewLogStore(db *sql.DB) ReviewLogStore { return &reviewLogStore{db} }

func (s *reviewLogStore) Create(log *ReviewLog) error {
	res, err := s.db.Exec(
		`INSERT INTO review_logs
		 (answer_id, self_rating, ease_factor_before, ease_factor_after, interval_before, interval_after)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		log.AnswerID, log.SelfRating,
		log.EaseFactorBefore, log.EaseFactorAfter,
		log.IntervalBefore, log.IntervalAfter,
	)
	if err != nil {
		return fmt.Errorf("review log create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("review log last insert id: %w", err)
	}
	log.ID = id
	return nil
}

func (s *reviewLogStore) FindByAnswerID(answerID int64) ([]ReviewLog, error) {
	rows, err := s.db.Query(
		`SELECT id, answer_id, self_rating,
		        ease_factor_before, ease_factor_after,
		        interval_before, interval_after, created_at
		 FROM review_logs WHERE answer_id = ? ORDER BY created_at`,
		answerID,
	)
	if err != nil {
		return nil, fmt.Errorf("review log find: %w", err)
	}
	defer rows.Close()

	var logs []ReviewLog
	for rows.Next() {
		var l ReviewLog
		var createdAt []byte
		if err := rows.Scan(
			&l.ID, &l.AnswerID, &l.SelfRating,
			&l.EaseFactorBefore, &l.EaseFactorAfter,
			&l.IntervalBefore, &l.IntervalAfter, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("review log scan: %w", err)
		}
		t, err := parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("review log parse created_at: %w", err)
		}
		l.CreatedAt = t
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
