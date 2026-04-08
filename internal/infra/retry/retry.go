package retry

import (
	"context"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func Do[T any](
	ctx context.Context,
	cfg Config,
	isRetryable func(error) bool,
	fn func() (T, error),
) (T, error) {

	var result T
	delay := cfg.BaseDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}

		if !isRetryable(err) {
			return result, err
		}

		if attempt == cfg.MaxAttempts {
			return result, err
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}

		if delay *= 2; delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}
	return result, nil
}
