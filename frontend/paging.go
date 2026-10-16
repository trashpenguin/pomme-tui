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
func (m model) helpPageCount() int {
	left, right := helpColumns()
	count := func(sections []helpSection) int {
		n := max(0, len(sections)-1)
		for _, s := range sections {
			n += 1 + len(s.entries)
		}
		return n
	}
	rows := max(count(left), count(right))
	size := max(1, m.h-6)
	return max(1, (rows+size-1)/size)
}
