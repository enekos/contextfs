package core

import (
	"encoding/json"
	"regexp"
	"strings"
)

var fenceRe = regexp.MustCompile("```(?:json)?\\s*|```")
var objectRe = regexp.MustCompile(`\{[\s\S]*\}`)
var arrayRe = regexp.MustCompile(`\[[\s\S]*\]`)

func stripFences(text string) string {
	return strings.TrimSpace(fenceRe.ReplaceAllString(text, ""))
}

func ExtractJSONObject(text string) map[string]any {
	match := objectRe.FindString(stripFences(text))
	if match == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(match), &out); err != nil {
		return nil
	}
	return out
}

func ExtractJSONArray(text string) []any {
	match := arrayRe.FindString(stripFences(text))
	if match == "" {
		return nil
	}
	var out []any
	if err := json.Unmarshal([]byte(match), &out); err != nil {
		return nil
	}
	return out
}
