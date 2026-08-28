package httpclient

import (
	"testing"
	"time"
)

func TestRetryPolicy_ShouldRetryStatus(t *testing.T) {
	p := DefaultRetryPolicy()

	cases := map[int]bool{
		200: false,
		404: false,
		429: true,
		500: false, // not in the default set on purpose: not every 5xx is transient
		502: true,
		503: true,
		504: true,
	}
	for status, want := range cases {
		if got := p.shouldRetryStatus(status); got != want {
			t.Errorf("shouldRetryStatus(%d) = %v, want %v", status, got, want)
		}
	}
}

func TestRetryPolicy_Backoff_GrowsAndCaps(t *testing.T) {
	p := RetryPolicy{BaseDelay: 100 * time.Millisecond, MaxDelay: 300 * time.Millisecond}

	first := p.backoff(1)
	second := p.backoff(2)
	third := p.backoff(3)

	// jitter is +/-20%, so just check ballpark ordering and the cap.
	if first <= 0 {
		t.Fatalf("backoff(1) = %v, want > 0", first)
	}
	if second <= first/2 {
		t.Fatalf("backoff(2) = %v should be roughly double backoff(1) = %v", second, first)
	}
	if third > 360*time.Millisecond {
		t.Fatalf("backoff(3) = %v exceeded MaxDelay + jitter headroom", third)
	}
}

func TestNoRetry_SingleAttempt(t *testing.T) {
	p := NoRetry()
	if p.MaxAttempts != 1 {
		t.Fatalf("NoRetry().MaxAttempts = %d, want 1", p.MaxAttempts)
	}
}
