package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"strings"
	"testing"
	"time"
)

func TestMiniPlayerHasWordmarkAndSeparateTrackDetails(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := mouseModel(120, 16, 0)
	m.mini = true
	m.st.Playing = true
	m.st.Now.Title = "Drown (New)"
	m.st.Now.Artist = "Bring Me The Horizon"
	m.st.Pos = 12 * time.Second
	m.st.Dur = 222 * time.Second
	view := m.View()
	for _, text := range []string{"POMME", "Drown (New)", "Bring Me The Horizon", "0:12", "3:42", "m expand"} {
		if !strings.Contains(view, text) {
			t.Fatalf("mini player missing %q", text)
		}
	}
	if strings.Contains(view, "Drown (New) - Bring Me The Horizon") {
		t.Fatal("mini title and artist still share one cramped line")
	}
}
func TestMiniPlayerFitsSizesAndLongUnicodeMetadata(t *testing.T) {
	for _, size := range [][2]int{{20, 5}, {30, 6}, {40, 8}, {80, 10}, {140, 14}, {200, 35}} {
		m := mouseModel(size[0], size[1], 0)
		m.mini = true
		m.st.Now.Title = strings.Repeat("長い曲名 ", 40)
		m.st.Now.Artist = strings.Repeat("Artist ", 30)
		m.note = strings.Repeat("Notice ", 30)
		m.st.Pos = 999 * time.Hour
		m.st.Dur = time.Second
		m.sleepUntil = time.Now().Add(time.Minute)
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > size[1] {
			t.Fatalf("mini height overflow at %v", size)
		}
		for _, line := range lines {
			if lipgloss.Width(line) > size[0] {
				t.Fatalf("mini width overflow at %v", size)
			}
		}
	}
}
func TestMiniPlayerShowsRecoveryAndInitialization(t *testing.T) {
	m := mouseModel(80, 12, 0)
	m.mini = true
	m.st.Playing = true
	m.st.Recovering = true
	if !strings.Contains(m.View(), "RECONNECTING") {
		t.Fatal("mini falsely labels recovery as playing")
	}
	m.st.Recovering = false
	m.beginAudioInitialization()
	if !strings.Contains(m.View(), "INITIALIZING") {
		t.Fatal("mini missing initialization state")
	}
}
func TestMiniPlayerFocusAndExpansion(t *testing.T) {
	m := mouseModel(120, 38, 0)
	m.focus = focusLibrary
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = next.(model)
	if !m.mini || m.focus != focusPlayer {
		t.Fatal("mini controls target hidden panel")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	if m.focus != focusPlayer {
		t.Fatal("mini Tab moved focus to hidden panel")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if next.(model).mini || next.(model).focus != focusLibrary {
		t.Fatal("expanding mini lost original focus")
	}
}

func TestMiniCardDoesNotStretchAcrossWideWindow(t *testing.T) {
	m := mouseModel(200, 35, 0)
	m.mini = true
	found := false
	for _, line := range strings.Split(m.View(), "\n") {
		if start := strings.Index(line, "╭"); start >= 0 {
			end := strings.Index(line, "╮")
			if end < start || lipgloss.Width(line[start:end+len("╮")]) > 90 {
				t.Fatal("mini card stretches across wide window")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("mini card border not rendered")
	}
}
