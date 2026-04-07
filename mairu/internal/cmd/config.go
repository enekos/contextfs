package cmd

import (
	"strings"
)

// GetAPIKey looks up the Gemini API key from the resolved config,
// falling back to the loaded config.
func GetAPIKey() string {
	if appConfig != nil && appConfig.API.GeminiAPIKey != "" {
		return cleanAPIKey(appConfig.API.GeminiAPIKey)
	}
	return ""
}

func cleanAPIKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, "\"'")
	return key
}
