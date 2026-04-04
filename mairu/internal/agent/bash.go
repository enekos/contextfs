package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

// RunBash executes a shell command with a timeout and returns its output.
func (a *Agent) RunBash(command string, timeoutMs int) (string, error) {
	if timeoutMs <= 0 {
		timeoutMs = 30000 // default 30s
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = a.db.Root() // Run in the project root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		if err := cmd.Process.Kill(); err != nil {
			return "", fmt.Errorf("command timed out and failed to kill: %w", err)
		}
		return "", fmt.Errorf("command timed out after %dms", timeoutMs)
	case err := <-done:
		outStr := stdout.String()
		errStr := stderr.String()

		result := ""
		if outStr != "" {
			result += "STDOUT:\n" + outStr
		}
		if errStr != "" {
			result += "\nSTDERR:\n" + errStr
		}

		if err != nil {
			result += fmt.Sprintf("\nExited with error: %v", err)
		}

		// Truncate if output is too long (e.g. max 5000 chars)
		if len(result) > 5000 {
			result = result[:5000] + "\n...[Output truncated]"
		}

		return result, nil
	}
}
