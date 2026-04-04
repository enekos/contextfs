package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

var ansiPattern = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func StripANSI(str string) string {
	return ansiPattern.ReplaceAllString(str, "")
}

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
		outStr := StripANSI(stdout.String())
		errStr := StripANSI(stderr.String())

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

		// Truncate if output is too long (max 10000 chars, tail truncation)
		if len(result) > 10000 {
			result = result[:10000] + "\n...[Output truncated, run command redirecting to file to see full output]"
		}

		return result, nil
	}
}
