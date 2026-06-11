package db

import (
	"context"
	"time"
)

// IncrementRateLimitBucket atomically increments a distributed rate-limit counter.
func (s *Store) IncrementRateLimitBucket(ctx context.Context, bucketKey string, windowStart time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO rate_limit_buckets (bucket_key, window_start, request_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (bucket_key, window_start)
		DO UPDATE SET request_count = rate_limit_buckets.request_count + 1
		RETURNING request_count`,
		bucketKey, windowStart.UTC(),
	).Scan(&count)
	return count, err
}

// CleanupRateLimitBuckets removes buckets older than the given cutoff.
func (s *Store) CleanupRateLimitBuckets(ctx context.Context, olderThan time.Time) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM rate_limit_buckets WHERE window_start < $1`,
		olderThan.UTC(),
	)
	return err
}
