package tui

import "testing"

func TestParseWorkspacePane(t *testing.T) {
	tests := []struct {
		input string
		want  workspacePane
		ok    bool
	}{
		{input: "agent", want: paneAgent, ok: true},
		{input: "chat", want: paneAgent, ok: true},
		{input: "nvim", want: paneNvim, ok: true},
		{input: "lazygit", want: paneLazygit, ok: true},
		{input: "git", want: paneLazygit, ok: true},
		{input: "unknown", want: "", ok: false},
	}

	for _, tc := range tests {
		got, ok := parseWorkspacePane(tc.input)
		if ok != tc.ok {
			t.Fatalf("parseWorkspacePane(%q) ok=%v, want %v", tc.input, ok, tc.ok)
		}
		if got != tc.want {
			t.Fatalf("parseWorkspacePane(%q) pane=%q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestPaneCommandSpec(t *testing.T) {
	tests := []struct {
		pane     workspacePane
		wantName string
		wantArgs int
		wantOK   bool
	}{
		{pane: paneNvim, wantName: "nvim", wantArgs: 0, wantOK: true},
		{pane: paneLazygit, wantName: "lazygit", wantArgs: 0, wantOK: true},
		{pane: paneAgent, wantName: "", wantArgs: 0, wantOK: false},
	}

	for _, tc := range tests {
		gotName, gotArgs, gotOK := paneCommandSpec(tc.pane)
		if gotOK != tc.wantOK {
			t.Fatalf("paneCommandSpec(%q) ok=%v, want %v", tc.pane, gotOK, tc.wantOK)
		}
		if gotName != tc.wantName {
			t.Fatalf("paneCommandSpec(%q) name=%q, want %q", tc.pane, gotName, tc.wantName)
		}
		if len(gotArgs) != tc.wantArgs {
			t.Fatalf("paneCommandSpec(%q) args=%d, want %d", tc.pane, len(gotArgs), tc.wantArgs)
		}
	}
}
