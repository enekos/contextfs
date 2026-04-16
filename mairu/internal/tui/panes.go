package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type externalToolDoneMsg struct {
	pane workspacePane
	err  error
}

type workspacePane string

const (
	paneAgent   workspacePane = "agent"
	paneNvim    workspacePane = "nvim"
	paneLazygit workspacePane = "lazygit"
)

func parseWorkspacePane(raw string) (workspacePane, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "agent", "chat":
		return paneAgent, true
	case "nvim", "vim":
		return paneNvim, true
	case "lazygit", "git":
		return paneLazygit, true
	default:
		return "", false
	}
}

func paneCommandSpec(p workspacePane) (name string, args []string, ok bool) {
	switch p {
	case paneNvim:
		return "nvim", nil, true
	case paneLazygit:
		return "lazygit", nil, true
	default:
		return "", nil, false
	}
}

func paneLabel(p workspacePane) string {
	switch p {
	case paneNvim:
		return "Neovim"
	case paneLazygit:
		return "LazyGit"
	default:
		return "Agent"
	}
}

func (m *model) openWorkspacePane(p workspacePane) tea.Cmd {
	m.activePane = p
	if p == paneAgent {
		m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Agent pane focused."})
		m.renderMessages()
		m.autoScroll()
		return nil
	}
	if m.thinking {
		m.activePane = paneAgent
		m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Wait for the current stream to finish before opening a workspace pane."})
		m.renderMessages()
		m.autoScroll()
		return nil
	}

	bin, args, ok := paneCommandSpec(p)
	if !ok {
		m.activePane = paneAgent
		return nil
	}
	if _, err := exec.LookPath(bin); err != nil {
		m.activePane = paneAgent
		m.messages = append(m.messages, ChatMessage{Role: "Error", Content: fmt.Sprintf("`%s` is not installed or not on PATH.", bin)})
		m.renderMessages()
		m.autoScroll()
		return nil
	}

	label := paneLabel(p)
	m.messages = append(m.messages, ChatMessage{
		Role:    "System",
		Content: fmt.Sprintf("Opening %s pane. Exit %s to return to Mairu.", label, label),
	})
	m.renderMessages()
	m.autoScroll()

	cmd := exec.Command(bin, args...)
	if m.agent != nil && m.agent.GetRoot() != "" {
		cmd.Dir = m.agent.GetRoot()
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return externalToolDoneMsg{pane: p, err: err}
	})
}
