package rate_limiter

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"time"
)

type rateLimit struct {
	Code             string
	BucketSize       int
	CurrentValue     int
	LastReservedAtMs int
}

type GormMySQLAdapter struct {
	db *gorm.DB
}

func (a *GormMySQLAdapter) Prepare(ctx context.Context) error {
	err := a.db.WithContext(ctx).Exec(`CREATE TABLE IF NOT EXISTS rate_limits (
    code VARCHAR(255) PRIMARY KEY,
    bucket_size INT,
    current_value INT DEFAULT 0,
    last_reserved_at_ms BIGINT DEFAULT NULL,
    CHECK (current_value <= bucket_size)
	)`).Error
	if err != nil {
		return fmt.Errorf("create rate_limits table: %w", err)
	}
	return nil
}

func (a *GormMySQLAdapter) ReserveToken(ctx context.Context, code string, bucketSize int, interval time.Duration) error {
	err := a.db.WithContext(ctx).Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&rateLimit{
		Code:             code,
		BucketSize:       bucketSize,
		LastReservedAtMs: int(time.Now().UnixMilli()),
	}).Error
	if err != nil {
		return fmt.Errorf("create default value: %w", err)
	}

	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row rateLimit
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "code = ?", code).Error
		if err != nil {
			return fmt.Errorf("fetching current value: %w", err)
		}

		newCountTokens := int(
			math.Floor(
				float64(time.Now().UnixMilli()-int64(row.LastReservedAtMs)) / float64(interval.Milliseconds()),
			),
		)
		if newCountTokens > bucketSize {
			newCountTokens = bucketSize
		}

		if newCountTokens < 1 {
			return NewReservationError(int(
				-(time.Now().UnixMilli() - int64(row.LastReservedAtMs) - interval.Milliseconds()),
			))
		}

		err = tx.Model(&rateLimit{}).Where("code = ?", row.Code).Updates(map[string]interface{}{
			"last_reserved_at_ms": time.Now().UnixMilli(),
			"current_value":       newCountTokens - 1,
		}).Error
		if err != nil {
			return fmt.Errorf("update to new calculated value: %w", err)
		}

		return nil
	})
}
