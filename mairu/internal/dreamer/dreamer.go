package dreamer

import "strings"

func BuildPrompt(goal string, context []string) string {
	if len(context) == 0 {
		return "Goal: " + goal
	}
	return "Goal: " + goal + "\nContext:\n- " + strings.Join(context, "\n- ")
}
