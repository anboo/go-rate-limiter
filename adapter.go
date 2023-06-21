package rate_limiter

import (
	"context"
	"fmt"
	"time"
)

type Adapter interface {
	Prepare(ctx context.Context) error
	ReserveToken(ctx context.Context, code string, bucketSize int, interval time.Duration) error
}

type ReservationError struct {
	NeedWaitMS int
}

func NewReservationError(needWaitNS int) *ReservationError {
	return &ReservationError{
		NeedWaitMS: needWaitNS,
	}
}

func (e ReservationError) Error() string {
	return fmt.Sprintf("need wait %d", e.NeedWaitMS)
}
