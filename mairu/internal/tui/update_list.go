package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleListUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var lsCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listModel.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.showList = false
			return m, nil
		case tea.KeyEnter:
			m.showList = false
			selectedItem := m.listModel.SelectedItem()
			if selectedItem != nil {
				if m.listType == "session" {
					sessionName := selectedItem.(listItem).title
					if m.sessionName != "" {
						_ = m.agent.SaveSession(m.sessionName)
					} else {
						_ = m.agent.SaveSession("current")
					}
					err := m.agent.LoadSession(sessionName)
					if err != nil {
						m.messages = append(m.messages, ChatMessage{Role: "Error", Content: "Failed to load session: " + err.Error()})
					} else {
						m.sessionName = sessionName
						m.messages = []ChatMessage{{Role: "System", Content: "Loaded session: " + sessionName}}
						found := false
						for _, s := range m.sessions {
							if s == sessionName {
								found = true
								break
							}
						}
						if !found {
							m.sessions = append(m.sessions, sessionName)
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
					m.viewport.GotoBottom()
				} else if m.listType == "model" {
					modelName := selectedItem.(listItem).title
					m.agent.SetModel(modelName)
					m.messages = append(m.messages, ChatMessage{Role: "System", Content: "Switched model to: " + modelName})
					m.renderMessages()
					m.viewport.GotoBottom()
				}
			}
			return m, nil
		}
	}
	m.listModel, lsCmd = m.listModel.Update(msg)
	return m, lsCmd
}
