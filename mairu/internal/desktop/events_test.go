package desktop

import (
	"mairu/internal/agent"
	"testing"
)

// Compile-time check for chat binding signatures.
var _ interface {
	ListSessions() ([]string, error)
	CreateSession(name string) error
	LoadSessionHistory(session string) ([]agent.SavedMessage, error)
	SendMessage(session, text string)
	ApproveAction(session string, approved bool)
} = (*App)(nil)

func TestChatBindingsCompile(t *testing.T) {
	t.Log("chat binding signatures verified")
}
