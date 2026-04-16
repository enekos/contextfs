package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.thinking {
			m.agent.Interrupt()
			m.messages = append(m.messages, ChatMessage{Role: "System", Content: "🛑 Interruption requested..."})
			m.renderMessages()
			m.autoScroll()
		} else {
			m.textarea.Reset()
		}
		return m, nil
	case tea.KeyCtrlC, tea.KeyCtrlD:
		if m.sessionName != "" {
			_ = m.agent.SaveSession(m.sessionName)
		}
		return m, tea.Quit
	case tea.KeyPgUp:
		m.viewport.HalfPageUp()
		m.followMode = false
		return m, nil
	case tea.KeyPgDown:
		m.viewport.HalfPageDown()
		m.followMode = false
		return m, nil
	case tea.KeyHome:
		m.viewport.GotoTop()
		m.followMode = false
		return m, nil
	case tea.KeyEnd:
		m.followMode = true
		m.viewport.GotoBottom()
		return m, nil
	case tea.KeyCtrlF:
		m.followMode = !m.followMode
		if m.followMode {
			m.viewport.GotoBottom()
			m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Follow mode enabled."})
		} else {
			m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Follow mode paused. Use End or Ctrl+F to resume."})
		}
		m.renderMessages()
		m.autoScroll()
		return m, nil
	case tea.KeyCtrlE:
		switch m.sidebarMode {
		case "session":
			m.sidebarMode = "explore"
			m.selectedMessage = clampMessageIndex(len(m.messages)-1, 0, len(m.messages))
			m.selectedEvent = -1
		case "explore":
			m.sidebarMode = "logs"
			if len(m.internalLogs) > 0 {
				m.selectedLog = len(m.internalLogs) - 1
			}
		default:
			m.sidebarMode = "session"
		}
		return m, nil
	case tea.KeyCtrlJ:
		if m.sidebarMode == "explore" {
			if m.selectedMessage >= 0 && m.selectedMessage < len(m.messages) {
				msg := m.messages[m.selectedMessage]
				if m.selectedEvent < len(msg.ToolEvents)-1 {
					m.selectedEvent++
				} else if m.selectedMessage < len(m.messages)-1 {
					m.selectedMessage++
					m.selectedEvent = -1
				}
			} else {
				m.selectedMessage = clampMessageIndex(m.selectedMessage, 1, len(m.messages))
				m.selectedEvent = -1
			}
			m.jumpToSelectedMessage()
			m.followMode = false
			m.renderMessages()
			return m, nil
		}
		if m.sidebarMode == "logs" {
			if len(m.internalLogs) > 0 {
				m.selectedLog = clampMessageIndex(m.selectedLog, 1, len(m.internalLogs))
			}
			return m, nil
		}
	case tea.KeyCtrlK:
		if m.sidebarMode == "explore" {
			if m.selectedEvent >= 0 {
				m.selectedEvent--
			} else if m.selectedMessage > 0 {
				m.selectedMessage--
				m.selectedEvent = len(m.messages[m.selectedMessage].ToolEvents) - 1
			} else {
				m.selectedMessage = clampMessageIndex(m.selectedMessage, -1, len(m.messages))
				m.selectedEvent = -1
			}
			m.jumpToSelectedMessage()
			m.followMode = false
			m.renderMessages()
			return m, nil
		}
		if m.sidebarMode == "logs" {
			if len(m.internalLogs) > 0 {
				m.selectedLog = clampMessageIndex(m.selectedLog, -1, len(m.internalLogs))
			}
			return m, nil
		}
	case tea.KeyCtrlP:
		models := []string{
			"gemini-3.1-flash-lite-preview",
			"gemini-3.1-pro-preview",
			"gemini-1.5-pro-latest",
			"gemini-1.5-flash-latest",
			"gemini-2.0-flash-exp",
			"gemini-2.0-pro-exp",
		}
		current := m.agent.GetModelName()
		idx := 0
		for i, mod := range models {
			if mod == current {
				idx = i
				break
			}
		}
		idx = (idx + 1) % len(models)
		newMod := models[idx]
		m.agent.SetModel(newMod)
		m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Switched model to: " + newMod})
		m.renderMessages()
		m.autoScroll()
		return m, nil
	case tea.KeyCtrlL:
		m.messages = []ChatMessage{{Role: "System", Content: "Terminal cleared."}}
		m.renderMessages()
		m.autoScroll()
		return m, nil
	case tea.KeyCtrlO:
		if len(m.sessions) > 1 {
			idx := -1
			for i, s := range m.sessions {
				if s == m.sessionName {
					idx = i
					break
				}
			}
			idx = (idx + 1) % len(m.sessions)
			newSession := m.sessions[idx]

			if m.sessionName != "" {
				_ = m.agent.SaveSession(m.sessionName)
			}
			err := m.agent.LoadSession(newSession)
			if err != nil {
				m.messages = append(m.messages, ChatMessage{Role: "Error", Content: "Failed to load session: " + err.Error()})
			} else {
				m.sessionName = newSession
				m.messages = []ChatMessage{{Role: "System", Content: "Loaded session: " + newSession}}
				found := false
				for _, s := range m.sessions {
					if s == newSession {
						found = true
						break
					}
				}
				if !found {
					m.sessions = append(m.sessions, newSession)
				}
				for _, text := range m.agent.GetHistoryText() {
					if strings.HasPrefix(text, "You: ") {
						m.messages = append(m.messages, ChatMessage{Role: "You", Content: strings.TrimPrefix(text, "You: ")})
					} else if strings.HasPrefix(text, "Mairu: ") {
						m.messages = append(m.messages, ChatMessage{Role: "Mairu", Content: strings.TrimPrefix(text, "Mairu: ")})
					}
				}
			}
			m.renderMessages()
			m.autoScroll()
		}
		return m, nil
	case tea.KeyCtrlN:
		return m, m.openWorkspacePane(paneNvim)
	case tea.KeyCtrlG:
		return m, m.openWorkspacePane(paneLazygit)
	}
	return m, nil
}
