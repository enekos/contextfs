package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRetryableStreamErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
		{name: "connection reset", err: errors.New("connection reset by peer"), want: true},
		{name: "hangup", err: errors.New("stream hangup"), want: true},
		{name: "broken pipe", err: errors.New("write: broken pipe"), want: true},
		{name: "resource exhausted", err: errors.New("RESOURCE_EXHAUSTED"), want: true},
		{name: "generic", err: errors.New("validation failed"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryableStreamErr(tc.err)
			if got != tc.want {
				t.Fatalf("isRetryableStreamErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestStreamRetryDelay(t *testing.T) {
	policy := RetryPolicy{
		BaseDelay: 200 * time.Millisecond,
		MaxDelay:  1500 * time.Millisecond,
	}
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 200 * time.Millisecond},
		{attempt: 2, want: 400 * time.Millisecond},
		{attempt: 3, want: 800 * time.Millisecond},
		{attempt: 4, want: 1500 * time.Millisecond},
	}
	for _, tc := range tests {
		got := streamRetryDelay(tc.attempt, policy)
		if got != tc.want {
			t.Fatalf("attempt %d delay = %s, want %s", tc.attempt, got, tc.want)
		}
	}
}
