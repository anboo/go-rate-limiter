package rate_limiter

import (
	"context"
	"fmt"
	"time"
)

type RateLimiter struct {
	adapter Adapter
}

func NewRateLimiter(adapter Adapter) *RateLimiter {
	return &RateLimiter{
		adapter: adapter,
	}
}

func (r *RateLimiter) ReserveOrWait(ctx context.Context, code string, bucketSize int, interval time.Duration) error {
	err := r.adapter.ReserveToken(ctx, code, bucketSize, interval)
	reservationError, ok := err.(ReservationError)
	if !ok {
		return fmt.Errorf("reservation error: %w", err)
	}

	duration := time.Duration(reservationError.NeedWaitMS) * time.Millisecond
	time.Sleep(duration)

	return r.ReserveOrWait(ctx, code, bucketSize, interval)
}
