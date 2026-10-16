package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
	"time"
)

func TestSearchHistoryBoundedAndDeduplicated(t *testing.T) {
	var history []string
	for i := 0; i < 100; i++ {
		history = rememberSearch(history, fmt.Sprint(i))
	}
	history = rememberSearch(history, "  99  ")
	if len(history) != 30 || history[0] != "99" || history[1] != "98" {
		t.Fatal(history)
	}
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	saveSearchHistory(history)
	if got := loadSearchHistory(); len(got) != 30 || got[0] != "99" {
		t.Fatal(got)
	}
}
func TestMiniPlayerFitsNarrowTerminal(t *testing.T) {
	m := model{mini: true, phase: phaseReady, w: 40, h: 8}
	m.st.Now.Title = strings.Repeat("long title ", 30)
	for _, line := range strings.Split(m.View(), "\n") {
		if lipgloss.Width(line) > 40 {
			t.Fatal("mini-player overflows")
		}
	}
}
func TestSleepTimerExpiresWhilePaused(t *testing.T) {
	m := model{sleepUntil: time.Now().Add(-time.Second)}
	next, _ := m.Update(tickMsg(time.Now()))
	if !next.(model).sleepUntil.IsZero() {
		t.Fatal("expired timer remains armed")
	}
}
func TestSleepTimerCanBeCanceled(t *testing.T) {
	m := model{phase: phaseReady, sleepChoice: 4, sleepUntil: time.Now().Add(time.Hour)}
	next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if !next.(model).sleepUntil.IsZero() {
		t.Fatal("timer not canceled")
	}
}
