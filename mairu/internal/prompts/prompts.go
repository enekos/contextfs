package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed *.md
var promptFiles embed.FS

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.ParseFS(promptFiles, "*.md"))
}

// Get renders a prompt template with the given data.
func Get(name string, data any) (string, error) {
	// 1. Try project-local override first
	cwd, _ := os.Getwd()
	localOverrides := []string{
		filepath.Join(cwd, ".mairu", "prompts", name+".md"),
		filepath.Join(cwd, "prompts", name+".md"),
	}
	for _, path := range localOverrides {
		if content, err := os.ReadFile(path); err == nil {
			t, err := template.New(name).Parse(string(content))
			if err != nil {
				return "", fmt.Errorf("failed to parse local override %s: %w", path, err)
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("failed to execute local override %s: %w", path, err)
			}
			return buf.String(), nil
		}
	}

	// 2. Try user-global override
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "mairu", "prompts", name+".md")
		if content, err := os.ReadFile(globalPath); err == nil {
			t, err := template.New(name).Parse(string(content))
			if err != nil {
				return "", fmt.Errorf("failed to parse global override %s: %w", globalPath, err)
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("failed to execute global override %s: %w", globalPath, err)
			}
			return buf.String(), nil
		}
	}

	// 3. Fallback to built-in template
	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, name+".md", data)
	if err != nil {
		return "", fmt.Errorf("failed to execute prompt template %s: %w", name, err)
	}
	return buf.String(), nil
}

// Render is a convenience function that panics on error, useful for static prompts or when you know the template is valid.
func Render(name string, data any) string {
	res, err := Get(name, data)
	if err != nil {
		panic(err)
	}
	return res
}
