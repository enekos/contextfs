package agent

import (
	"errors"
	"testing"
)

func TestIsRetryableStreamErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "connection reset", err: errors.New("connection reset by peer"), want: true},
		{name: "hangup", err: errors.New("stream hangup"), want: true},
		{name: "broken pipe", err: errors.New("write: broken pipe"), want: true},
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
