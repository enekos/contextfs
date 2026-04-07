package cmd

import "fmt"

// CLIError is a user-facing error with an actionable hint.
type CLIError struct {
	Message string
	Hint    string
	Cause   error
}

func (e *CLIError) Error() string {
	s := "Error: " + e.Message
	if e.Hint != "" {
		s += "\n  Hint: " + e.Hint
	}
	s += "\n  Run 'mairu doctor' for a full diagnostic"
	return s
}

func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewCLIError creates a CLIError with a formatted message.
func NewCLIError(cause error, hint string, format string, args ...any) *CLIError {
	return &CLIError{
		Message: fmt.Sprintf(format, args...),
		Hint:    hint,
		Cause:   cause,
	}
}
