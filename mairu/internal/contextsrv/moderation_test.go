package contextsrv

import "testing"

func TestModerateContent(t *testing.T) {
	t.Run("clean content passes", func(t *testing.T) {
		res := ModerateContent("normal project note without secrets")
		if res.Status != ModerationStatusClean {
			t.Fatalf("expected clean, got %s", res.Status)
		}
		if len(res.Reasons) != 0 {
			t.Fatalf("expected no reasons, got %v", res.Reasons)
		}
	})

	t.Run("soft flags for suspicious text", func(t *testing.T) {
		res := ModerateContent("this includes password=supersecret")
		if res.Status != ModerationStatusFlaggedSoft {
			t.Fatalf("expected flagged_soft, got %s", res.Status)
		}
		if len(res.Reasons) == 0 {
			t.Fatalf("expected at least one reason")
		}
	})

	t.Run("hard reject for critical private key", func(t *testing.T) {
		res := ModerateContent("-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----")
		if res.Status != ModerationStatusRejectHard {
			t.Fatalf("expected reject_hard, got %s", res.Status)
		}
	})
}
