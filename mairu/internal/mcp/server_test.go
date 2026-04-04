package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServer(t *testing.T) {
	handler := NewServer()
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"ok":true,"service":"mcp"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
