package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderSidebar() string {
	stats := computeSessionStats(m.messages, m.currentResponse, m.toolEvents, m.thinking, m.agent.GetModelName())
	if m.sidebarMode == "explore" {
		return m.renderExploreSidebar(stats)
	}

	var sb strings.Builder
	sb.WriteString(sidebarHeaderStyle.Render("Session"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", sidebarLabelStyle.Render("Model:"), stats.Model))
	sb.WriteString(fmt.Sprintf("%s %s\n", sidebarLabelStyle.Render("State:"), stats.StreamState))
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("Messages:"), len(m.messages)))
	sb.WriteString(fmt.Sprintf("%s U:%d A:%d S:%d E:%d D:%d\n",
		sidebarLabelStyle.Render("By role:"),
		stats.UserMessages,
		stats.AssistantMessages,
		stats.SystemMessages,
		stats.ErrorMessages,
		stats.DiffMessages,
	))
	sb.WriteString("\n")
	sb.WriteString(sidebarHeaderStyle.Render("Tools"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("Events:"), stats.ToolEvents))
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("Calls:"), stats.ToolCalls))
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("Results:"), stats.ToolResults))
	sb.WriteString("\n")
	sb.WriteString(sidebarHeaderStyle.Render("Token Estimate"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("User:"), stats.EstimatedUserTokens))
	sb.WriteString(fmt.Sprintf("%s %d\n", sidebarLabelStyle.Render("Mairu:"), stats.EstimatedAgentTokens))
	sb.WriteString(fmt.Sprintf("%s %d / %dk\n", sidebarLabelStyle.Render("Total:"), stats.EstimatedTotalTokens, stats.ContextLimit/1000))

	percent := float64(stats.EstimatedTotalTokens) / float64(stats.ContextLimit) * 100
	bar := renderProgressBar(stats.EstimatedTotalTokens, stats.ContextLimit, 20)
	sb.WriteString(fmt.Sprintf("%s %s %.1f%%\n", sidebarLabelStyle.Render("Usage:"), bar, percent))

	if len(m.toolEvents) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(sidebarHeaderStyle.Render("Recent Tool Activity"))
		sb.WriteString("\n")
		start := len(m.toolEvents) - 5
		if start < 0 {
			start = 0
		}
		for _, e := range m.toolEvents[start:] {
			sb.WriteString("• " + previewText(e.Title, 44) + "\n")
		}
	}

	return sb.String()
}

func (m model) renderExploreSidebar(stats sessionStats) string {
	var sb strings.Builder
	sb.WriteString(sidebarHeaderStyle.Render("Explore"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %d / %d\n", sidebarLabelStyle.Render("Selected:"), m.selectedMessage+1, len(m.messages)))
	sb.WriteString(fmt.Sprintf("%s %s\n\n", sidebarLabelStyle.Render("Follow mode:"), boolLabel(m.followMode)))

	sb.WriteString(sidebarHeaderStyle.Render("Messages"))
	sb.WriteString("\n")
	start := len(m.messages) - 12
	if start < 0 {
		start = 0
	}
	for i := start; i < len(m.messages); i++ {
		msg := m.messages[i]
		prefix := " "
		if i == m.selectedMessage && m.selectedEvent == -1 {
			prefix = ">"
		} else if i == m.selectedMessage {
			prefix = " "
		}

		roleStr := msg.Role
		if roleStr == "Mairu" && len(msg.ToolEvents) > 0 {
			roleStr += fmt.Sprintf(" [%d]", len(msg.ToolEvents))
		}

		sb.WriteString(fmt.Sprintf("%s #%d %-10s %s\n",
			prefix,
			i+1,
			roleStr,
			previewText(msg.Content, 20),
		))

		if i == m.selectedMessage && len(msg.ToolEvents) > 0 {
			for j, e := range msg.ToolEvents {
				evPrefix := "   "
				if m.selectedEvent == j {
					evPrefix = " > "
				}
				sb.WriteString(fmt.Sprintf("%s %s\n", evPrefix, previewText(e.Title, 30)))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(sidebarHeaderStyle.Render("Tool Drilldown"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s events:%d calls:%d results:%d\n",
		sidebarLabelStyle.Render("Stats:"),
		stats.ToolEvents,
		stats.ToolCalls,
		stats.ToolResults,
	))
	logStart := len(m.toolLog) - 6
	if logStart < 0 {
		logStart = 0
	}
	for _, entry := range m.toolLog[logStart:] {
		sb.WriteString("• " + previewText(entry, 60) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(sidebarLabelStyle.Render("Ctrl+J/Ctrl+K navigate  ·  /jump <n>"))
	return sb.String()
}

func boolLabel(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func renderProgressBar(current, total, width int) string {
	if width <= 0 {
		return ""
	}
	if total <= 0 {
		total = 1
	}
	percent := float64(current) / float64(total)
	if percent > 1.0 {
		percent = 1.0
	}
	filledChars := int(float64(width) * percent)
	emptyChars := width - filledChars
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#88c0d0")).Render(strings.Repeat("█", filledChars)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#4c566a")).Render(strings.Repeat("░", emptyChars))
}
