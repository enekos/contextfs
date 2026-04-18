package redact

import (
	"strings"
	"testing"
)

// contains is a local helper so test failure messages read naturally.
func contains(haystack, needle string) bool {
	return len(needle) > 0 && strings.Contains(haystack, needle)
}

func TestNewAppliesDefaults(t *testing.T) {
	r := New()
	if r.entropyThreshold == 0 {
		t.Fatal("expected non-zero default entropyThreshold")
	}
	if r.damageCapRatio == 0 {
		t.Fatal("expected non-zero default damageCapRatio")
	}
	if r.minEntropyLen == 0 {
		t.Fatal("expected non-zero default minEntropyLen")
	}
}

func TestNewAcceptsOptions(t *testing.T) {
	r := New(
		WithEntropyThreshold(5.0),
		WithDamageCapRatio(0.75),
		WithMinEntropyLen(32),
		WithDenylistCommands([]string{"vault"}),
	)
	if r.entropyThreshold != 5.0 {
		t.Errorf("entropyThreshold = %v; want 5.0", r.entropyThreshold)
	}
	if r.damageCapRatio != 0.75 {
		t.Errorf("damageCapRatio = %v; want 0.75", r.damageCapRatio)
	}
	if r.minEntropyLen != 32 {
		t.Errorf("minEntropyLen = %v; want 32", r.minEntropyLen)
	}
	if len(r.denylistCommands) != 1 || r.denylistCommands[0] != "vault" {
		t.Errorf("denylistCommands = %v; want [vault]", r.denylistCommands)
	}
}

func TestRedactEmptyInputIsSafe(t *testing.T) {
	got := New().Redact("", KindText)
	if got.Redacted != "" {
		t.Errorf("Redacted = %q; want empty", got.Redacted)
	}
	if !got.EmbeddingSafe {
		t.Error("empty input must be embedding-safe")
	}
	if got.Dropped {
		t.Error("empty input must not be dropped")
	}
	if len(got.Findings) != 0 {
		t.Errorf("len(Findings) = %d; want 0", len(got.Findings))
	}
}

func TestRedactPlainTextPassesThrough(t *testing.T) {
	got := New().Redact("hello world this is fine", KindText)
	if got.Redacted != "hello world this is fine" {
		t.Errorf("Redacted = %q; want pass-through", got.Redacted)
	}
	if !got.EmbeddingSafe {
		t.Error("plain text must be embedding-safe")
	}
}

func TestLayer1RedactsGitHubPAT(t *testing.T) {
	in := "token=ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "ghp_1234567890") {
		t.Errorf("raw PAT leaked: %q", got.Redacted)
	}
	if got.EmbeddingSafe {
		t.Error("Layer 1 hit must set EmbeddingSafe=false")
	}
	if len(got.Findings) == 0 || got.Findings[0].Layer != LayerKnownToken {
		t.Errorf("expected LayerKnownToken finding; got %+v", got.Findings)
	}
}

func TestLayer1RedactsAWSAccessKey(t *testing.T) {
	in := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key leaked: %q", got.Redacted)
	}
	if got.EmbeddingSafe {
		t.Error("AWS key must set EmbeddingSafe=false")
	}
}

func TestLayer1RedactsStripeLiveKey(t *testing.T) {
	in := "sk_live_4eC39HqLyjWDarjtT1zdp7dcABCDEFGH"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "sk_live_4eC39HqLyjWDarjtT1zdp7dc") {
		t.Errorf("stripe key leaked: %q", got.Redacted)
	}
	if got.EmbeddingSafe {
		t.Error("stripe key must set EmbeddingSafe=false")
	}
}

func TestLayer1RedactsSlackToken(t *testing.T) {
	in := "xoxb-1234567890-0987654321-AbCdEfGhIjKlMnOpQrStUvWx"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "xoxb-1234567890") {
		t.Errorf("slack token leaked: %q", got.Redacted)
	}
}

func TestLayer1RedactsJWT(t *testing.T) {
	in := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Errorf("JWT leaked: %q", got.Redacted)
	}
}

func TestLayer1RedactsURICredentials(t *testing.T) {
	in := "postgres://admin:hunter2supersecret@db.internal:5432/app"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "hunter2supersecret") {
		t.Errorf("URI password leaked: %q", got.Redacted)
	}
}

func TestLayer1RedactsPEMPrivateKey(t *testing.T) {
	in := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\n-----END RSA PRIVATE KEY-----"
	got := New().Redact(in, KindText)
	if contains(got.Redacted, "MIIEpAIBAAKCAQEA") {
		t.Errorf("PEM body leaked: %q", got.Redacted)
	}
}
