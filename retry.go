package codex

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// RetryOptions configures RetryOnOverload.
type RetryOptions struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	JitterRatio  float64
}

// DefaultRetryOptions matches the reference SDK retry behavior.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 250 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		JitterRatio:  0.2,
	}
}

// RetryOnOverload retries op only for transient overload errors.
func RetryOnOverload[T any](ctx context.Context, options RetryOptions, op func(context.Context) (T, error)) (T, error) {
	return retryOnOverload(ctx, options, op, realRetryWait, rand.Float64)
}

type retryWait func(context.Context, time.Duration) error
type retryRand func() float64

func retryOnOverload[T any](ctx context.Context, options RetryOptions, op func(context.Context) (T, error), wait retryWait, random retryRand) (T, error) {
	var zero T
	if ctx == nil {
		return zero, errors.New("codex: nil context")
	}
	if op == nil {
		return zero, errors.New("codex: nil retry operation")
	}
	if wait == nil {
		return zero, errors.New("codex: nil retry wait function")
	}
	if random == nil {
		return zero, errors.New("codex: nil retry random source")
	}
	if options.MaxAttempts < 1 {
		return zero, errors.New("codex: max attempts must be at least one")
	}
	if options.InitialDelay < 0 || options.MaxDelay < 0 {
		return zero, errors.New("codex: retry delays must not be negative")
	}
	if options.JitterRatio < 0 {
		return zero, errors.New("codex: jitter ratio must not be negative")
	}

	delay := options.InitialDelay
	for attempt := 1; attempt <= options.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		value, err := op(ctx)
		if err == nil {
			return value, nil
		}
		if attempt == options.MaxAttempts || !IsRetryable(err) {
			return zero, err
		}

		capped := min(delay, options.MaxDelay)
		jitter := float64(delay) * options.JitterRatio
		waitFor := time.Duration(float64(capped) + ((random()*2)-1)*jitter)
		if waitFor > 0 {
			if err := wait(ctx, waitFor); err != nil {
				return zero, err
			}
		}
		delay = min(options.MaxDelay, delay*2)
	}
	return zero, errors.New("codex: retry loop exhausted")
}

func realRetryWait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
