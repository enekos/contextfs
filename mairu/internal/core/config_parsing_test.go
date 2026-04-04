package core

import "testing"

func TestParseBoolEnv(t *testing.T) {
	if !ParseBoolEnv("true", false) {
		t.Fatal("expected true")
	}
	if ParseBoolEnv("invalid", true) != true {
		t.Fatal("expected fallback true")
	}
}

func TestParseIntEnv(t *testing.T) {
	if ParseIntEnv("42", 1) != 42 {
		t.Fatal("expected parsed int")
	}
	if ParseIntEnv("x", 7) != 7 {
		t.Fatal("expected fallback")
	}
}
