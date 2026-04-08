package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestMapCmd(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	mapCmd.Run(mapCmd, []string{"."})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out == "" {
		t.Errorf("mapCmd output is empty")
	}
}

func TestSysCmd(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	sysCmd.Run(sysCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out == "" {
		t.Errorf("sysCmd output is empty")
	}
}

func TestInfoCmd(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	infoCmd.Run(infoCmd, []string{"."})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out == "" {
		t.Errorf("infoCmd output is empty")
	}
}

func TestEnvCmd(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "testenv*.env")
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("API_KEY=secret_hash_here\nexport PORT=8080\nDEBUG=true\nURL=http://localhost:3000\n# comment\n")
	tmpFile.Close()

	// Run 1: Normal mode (just keys)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	envReveal = false
	envPattern = ""
	envCmd.Run(envCmd, []string{tmpFile.Name()})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out == "" {
		t.Errorf("envCmd output is empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("API_KEY")) || !bytes.Contains(buf.Bytes(), []byte("PORT")) {
		t.Errorf("envCmd output missing keys: %s", out)
	}

	// Run 2: Reveal mode
	r, w, _ = os.Pipe()
	os.Stdout = w
	envReveal = true
	envCmd.Run(envCmd, []string{tmpFile.Name()})
	w.Close()
	os.Stdout = oldStdout
	var buf2 bytes.Buffer
	buf2.ReadFrom(r)
	out2 := buf2.String()

	if !bytes.Contains(buf2.Bytes(), []byte(`"val":"true"`)) {
		t.Errorf("envCmd reveal failed for safe boolean: %s", out2)
	}
	if !bytes.Contains(buf2.Bytes(), []byte(`"val":"8080"`)) {
		t.Errorf("envCmd reveal failed for safe short string: %s", out2)
	}
	if bytes.Contains(buf2.Bytes(), []byte("secret_hash_here")) {
		t.Errorf("envCmd LEAKED A SECRET: %s", out2)
	}
}
