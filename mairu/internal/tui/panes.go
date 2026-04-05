package tui

import "strings"

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
