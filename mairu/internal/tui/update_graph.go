package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleGraphUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var lsCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dataExplorer.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.showGraph = false
			return m, nil
		case tea.KeyEnter:
			selected := m.dataExplorer.lists[m.dataExplorer.activeTab].SelectedItem()
			if selected != nil {
				if graphItem, ok := selected.(graphListItem); ok {
					m.showGraph = false
					m.messages = append(m.messages, ChatMessage{Role: "System", Content: fmt.Sprintf("**Graph Node: %s**\n\n%s\n\n```\n%s\n```", graphItem.uri, graphItem.desc, graphItem.content)})
					m.renderMessages()
					m.viewport.GotoBottom()
				}
			}
			return m, nil
		}
	}
	m.dataExplorer, lsCmd = m.dataExplorer.Update(msg)
	return m, lsCmd
}
