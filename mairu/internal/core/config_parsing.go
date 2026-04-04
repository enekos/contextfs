package core

import (
	"strconv"
	"strings"
)

func ParseBoolEnv(raw string, fallback bool) bool {
	trim := strings.TrimSpace(strings.ToLower(raw))
	if trim == "" {
		return fallback
	}
	switch trim {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func ParseIntEnv(raw string, fallback int) int {
	trim := strings.TrimSpace(raw)
	if trim == "" {
		return fallback
	}
	v, err := strconv.Atoi(trim)
	if err != nil {
		return fallback
	}
	return v
}
