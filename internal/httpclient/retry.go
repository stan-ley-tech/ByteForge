package httpclient

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"time"
)

// RetryPolicy controls whether and how a Client retries a request.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts, including the first.
	// A value of 1 (or 0) disables retries.
	MaxAttempts int

	// BaseDelay is the delay before the second attempt. Each subsequent
	// attempt doubles it (exponential backoff), capped at MaxDelay.
	BaseDelay time.Duration
	MaxDelay  time.Duration

	// RetryStatusCodes lists response status codes that should trigger a
	// retry (typically 429 and the 5xx range). A response whose status
	// isn't in this set is treated as final, success or not.
	RetryStatusCodes map[int]bool
}

// DefaultRetryPolicy retries connection-level failures and common transient
// server responses with capped exponential backoff and jitter.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		RetryStatusCodes: map[int]bool{
			429: true,
			502: true,
			503: true,
			504: true,
		},
	}
}

// NoRetry disables retries entirely: a single attempt, whatever it returns.
func NoRetry() RetryPolicy {
	return RetryPolicy{MaxAttempts: 1}
}

func (p RetryPolicy) shouldRetryStatus(code int) bool {
	return p.RetryStatusCodes[code]
}

// shouldRetryError decides whether a transport-level error is worth retrying.
// Timeouts and connection resets are; anything wrapping context.Canceled is
// not, since the caller has already given up.
func (p RetryPolicy) shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || isConnectionReset(err)
	}
	return isConnectionReset(err)
}

func isConnectionReset(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

// wait blocks for the backoff delay of the given attempt (1-indexed retry
// count) or returns early with ctx's error if it's cancelled first.
func (p RetryPolicy) wait(ctx context.Context, attempt int) error {
	delay := p.backoff(attempt)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// backoff computes exponential delay with +/-20% jitter so that many
// concurrent requests retrying the same failing host don't all land on the
// same instant (the "thundering herd" problem).
func (p RetryPolicy) backoff(attempt int) time.Duration {
	if p.BaseDelay <= 0 {
		return 0
	}

	factor := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(p.BaseDelay) * factor)
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		delay = p.MaxDelay
	}

	jitterRange := float64(delay) * 0.2
	jitter := time.Duration((rand.Float64()*2 - 1) * jitterRange) //nolint:gosec // jitter, not security sensitive
	return delay + jitter
}
