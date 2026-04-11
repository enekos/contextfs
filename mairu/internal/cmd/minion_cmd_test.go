package cmd

import "testing"

func TestNewMinionCmd_RegistersCouncilFlag(t *testing.T) {
	minionCouncil = false
	c := NewMinionCmd()
	f := c.Flags().Lookup("council")
	if f == nil {
		t.Fatalf("expected --council flag to be registered")
	}
}

func TestNewMinionCmd_RegistersPRReviewOnlyFlag(t *testing.T) {
	minionPRReviewOnly = false
	c := NewMinionCmd()
	f := c.Flags().Lookup("pr-review-only")
	if f == nil {
		t.Fatalf("expected --pr-review-only flag to be registered")
	}
}
