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

	p := GetParser()
	if p == nil {
		t.Fatal("expected parser")
	}
	PutParser(p)
}
