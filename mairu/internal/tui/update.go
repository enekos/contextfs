package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && len(m.filteredCommands) > 0 {
		switch keyMsg.Type {
		case tea.KeyUp:
			m.autocompleteIndex--
			if m.autocompleteIndex < 0 {
				m.autocompleteIndex = len(m.filteredCommands) - 1
			}
			return m, nil
		case tea.KeyDown:
			m.autocompleteIndex++
			if m.autocompleteIndex >= len(m.filteredCommands) {
				m.autocompleteIndex = 0
			}
			return m, nil
		case tea.KeyTab:
			m.textarea.SetValue(m.filteredCommands[m.autocompleteIndex].Name + " ")
			m.textarea.SetCursor(len(m.textarea.Value()))
			m.filteredCommands = nil
			return m, nil
		case tea.KeyEsc:
			m.filteredCommands = nil
			return m, nil
		}
	}

	if m.showList {
		return m.handleListUpdate(msg)
	}
	if m.showGraph {
		return m.handleGraphUpdate(msg)
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
		cmds  []tea.Cmd
	)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd, spCmd)

	if _, ok := msg.(tea.KeyMsg); ok {
		val := m.textarea.Value()
		if strings.HasPrefix(val, "/") {
			m.filteredCommands = nil
			for _, cmd := range allSlashCommands {
				if strings.HasPrefix(cmd.Name, val) {
					m.filteredCommands = append(m.filteredCommands, cmd)
				}
			}
			if m.autocompleteIndex >= len(m.filteredCommands) {
				m.autocompleteIndex = 0
			}
		} else {
			m.filteredCommands = nil
		}
	}

	switch msg := msg.(type) {
	case deleteItemMsg:
		return m.handleDeleteItemMsg(msg)
	case animTickMsg:
		m2, cmd, done := m.handleAnimTickMsg(msg)
		if done {
			return m2, cmd
		}
		m = m2
		cmds = append(cmds, cmd)
	case spinner.TickMsg:
		m = m.handleSpinnerTickMsg(msg)
	case tea.WindowSizeMsg:
		m = m.handleWindowSizeMsg(msg)
	case tea.MouseMsg:
		m = m.handleMouseMsg(msg)
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			m2, cmd, done := m.handleEnter(msg)
			if done {
				return m2, cmd
			}
			m = m2
			cmds = append(cmds, cmd)
		} else {
			return m.handleKeyMsg(msg)
		}
	case agentStreamMsg:
		if cmd := m.handleAgentStream(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case errMsg:
		m.err = msg
		return m, nil
	case externalToolDoneMsg:
		return m.handleExternalToolDoneMsg(msg)
	}

	lines := strings.Count(m.textarea.Value(), "\n") + 1
	if lines > 5 {
		lines = 5
	}
	if m.textarea.Height() != lines {
		m.textarea.SetHeight(lines)
		m.recomputeLayout()
		m.autoScroll()
	}

	return m, tea.Batch(cmds...)
}
