package tui

import "strings"

func clampMessageIndex(current, delta, total int) int {
	if total <= 0 {
		return -1
	}
	if current < 0 {
		if delta >= 0 {
			return 0
		}
		return total - 1
	}
	next := current + delta
	if next < 0 {
		return 0
	}
	if next >= total {
		return total - 1
	}
	return next
}

func previewText(content string, maxLen int) string {
	normalized := strings.ReplaceAll(content, "\n", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	if maxLen <= 0 || len(normalized) <= maxLen {
		return normalized
	}
	if maxLen <= 3 {
		return normalized[:maxLen]
	}
	return normalized[:maxLen-3] + "..."
}

func previewMultiline(content string, maxLen int) string {
	trimmed := strings.TrimSpace(content)
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}
