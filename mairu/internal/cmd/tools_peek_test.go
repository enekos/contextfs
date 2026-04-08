package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestPeekCmd(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "testpeek*.txt")
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("line 1\nfunc myTestFunc() {\n  return 1\n}\nline 5\n")
	tmpFile.Close()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	peekLines = ""
	peekSymbol = "myTestFunc"

	peekCmd.Run(peekCmd, []string{tmpFile.Name()})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out == "" {
		t.Errorf("peekCmd output is empty")
	}
}
