package cmd

import (
	"strings"
	"testing"
)

func TestNewMinionCmd_RegistersCouncilFlag(t *testing.T) {
	minionCouncil = false
	c := NewMinionCmd()
	f := c.Flags().Lookup("council")
	if f == nil {
		t.Fatalf("expected --council flag to be registered")
	}
}

func TestFormatReviewerFindings_IncludesStructuredStatus(t *testing.T) {
	got := formatReviewerFindings(map[string]prReviewerOutcome{
		"App Developer": {
			role:    "App Developer",
			status:  reviewerStatusOK,
			content: "Looks safe",
		},
		"Tests Evangelist": {
			role:   "Tests Evangelist",
			status: reviewerStatusFailed,
			err:    "provider timeout",
		},
	})

	if !strings.Contains(got, "status: ok") {
		t.Fatalf("expected ok status in output")
	}
	if !strings.Contains(got, "status: failed") {
		t.Fatalf("expected failed status in output")
	}
	if !strings.Contains(got, "failure_reason: provider timeout") {
		t.Fatalf("expected failure reason in output")
	}
}
