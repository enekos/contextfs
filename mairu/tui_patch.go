package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func foo(msg tea.KeyMsg) {
	_ = msg.Alt
}
