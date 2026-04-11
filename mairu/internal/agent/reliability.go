package agent

import (
	"context"
	"errors"
	"strings"
	"time"
)

type RetryPolicy struct {
	MaxAttempts    int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	AttemptTimeout time.Duration
}

type ReliabilityConfig struct {
	StreamRetry       RetryPolicy
	CouncilTimeout    time.Duration
	CompactionTimeout time.Duration
}

func (c ReliabilityConfig) withDefaults() ReliabilityConfig {
	out := c
	if out.StreamRetry.MaxAttempts <= 0 {
		out.StreamRetry.MaxAttempts = 2
	}
	if out.StreamRetry.BaseDelay <= 0 {
		out.StreamRetry.BaseDelay = 750 * time.Millisecond
	}
	if out.StreamRetry.MaxDelay <= 0 {
		out.StreamRetry.MaxDelay = 5 * time.Second
	}
	if out.StreamRetry.AttemptTimeout <= 0 {
		out.StreamRetry.AttemptTimeout = 8 * time.Minute
	}
	if out.CouncilTimeout <= 0 {
		out.CouncilTimeout = 90 * time.Second
	}
	if out.CompactionTimeout <= 0 {
		out.CompactionTimeout = 45 * time.Second
	}
	return out
}

func DefaultReliabilityConfig() ReliabilityConfig {
	return ReliabilityConfig{}.withDefaults()
}

func streamRetryDelay(attempt int, policy RetryPolicy) time.Duration {
	if attempt <= 1 {
		return policy.BaseDelay
	}
	delay := policy.BaseDelay << uint(attempt-1)
	if delay > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableStreamErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "hangup") ||
		strings.Contains(lower, "sighup") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "stream removed") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "temporarily unavailable") ||
		strings.Contains(lower, "resource exhausted")
}
