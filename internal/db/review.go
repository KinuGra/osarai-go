package db

import (
	"database/sql"
	"fmt"
	"time"
)

type reviewStore struct{ db *sql.DB }

func NewReviewStore(db *sql.DB) ReviewStore { return &reviewStore{db} }

func (s *reviewStore) Create(review *Review) error {
	res, err := s.db.Exec(
		`INSERT INTO reviews (answer_id, ease_factor, interval_days, repetitions, next_review_at)
		 VALUES (?, ?, ?, ?, ?)`,
		review.AnswerID, review.EaseFactor, review.IntervalDays,
		review.Repetitions, formatDate(review.NextReviewAt),
	)
	if err != nil {
		return fmt.Errorf("review create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("review last insert id: %w", err)
	}
	review.ID = id
	return nil
}

func (s *reviewStore) FindDue(now time.Time) ([]Review, error) {
	rows, err := s.db.Query(
		`SELECT id, answer_id, ease_factor, interval_days, repetitions, next_review_at, updated_at
		 FROM reviews WHERE next_review_at <= ?
		 ORDER BY next_review_at`,
		formatDate(now),
	)
	if err != nil {
		return nil, fmt.Errorf("review find due: %w", err)
	}
	defer rows.Close()
	return scanReviews(rows)
}

func (s *reviewStore) Update(review *Review) error {
	_, err := s.db.Exec(
		`UPDATE reviews
		 SET ease_factor    = ?,
		     interval_days  = ?,
		     repetitions    = ?,
		     next_review_at = ?,
		     updated_at     = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		 WHERE id = ?`,
		review.EaseFactor, review.IntervalDays, review.Repetitions,
		formatDate(review.NextReviewAt), review.ID,
	)
	return err
}

func scanReviews(rows *sql.Rows) ([]Review, error) {
	var reviews []Review
	for rows.Next() {
		var r Review
		var nextReviewAt, updatedAt []byte
		if err := rows.Scan(
			&r.ID, &r.AnswerID, &r.EaseFactor, &r.IntervalDays,
			&r.Repetitions, &nextReviewAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		if t, err := parseTime(nextReviewAt); err == nil {
			r.NextReviewAt = t
		}
		if t, err := parseTime(updatedAt); err == nil {
			r.UpdatedAt = t
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}
