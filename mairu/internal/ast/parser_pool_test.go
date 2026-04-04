package ast

import "testing"

func TestParserPoolLifecycle(t *testing.T) {
	DeleteParserPool()
	if ParserPoolInitialized() {
		t.Fatal("expected uninitialized")
	}
	InitParserPool()
	if !ParserPoolInitialized() {
		t.Fatal("expected initialized")
	}
}
