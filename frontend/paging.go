package main

import tea "github.com/charmbracelet/bubbletea"

// Ctrl+B/F are alternatives when the terminal intercepts Page Up/Down.
func pageDirection(msg tea.KeyMsg) int {
	switch msg.String() {
	case "pgup", "ctrl+b":
		return -1
	case "pgdown", "ctrl+f":
		return 1
	}
	return 0
}
func (m model) searchRows() int { return max(1, min(12, m.h-12)) }
