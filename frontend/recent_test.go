package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func demoRecent(n int) []engine.Track {
	out := make([]engine.Track, n)
	for i := range out {
		out[i] = engine.Track{
			ID:     string(rune('a' + i)),
			Kind:   "album",
			Title:  "Album With A Fairly Long Name",
			Artist: "Some Artist",
			Art:    "https://example.invalid/{w}x{h}.jpg",
		}
	}
	return out
}

func recentModel() model {
	return model{
		w: 120, h: 35, phase: phaseReady, st: demoState(),
		recent: demoRecent(7),
	}
}

// Tab must walk queue → recent → player and wrap back.
func TestTabCyclesThreeZones(t *testing.T) {
	m := recentModel()
	want := []int{focusRecent, focusPlayer, focusQueue}
	for i, w := range want {
		next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(model)
		if m.focus != w {
			t.Fatalf("tab #%d put focus on %d, want %d", i+1, m.focus, w)
		}
	}
}

// With no recently-played entries, or in a terminal too short to draw the grid,
// Tab has to skip the zone rather than focus something invisible.
func TestTabSkipsHiddenRecentGrid(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    model
	}{
		{"no entries", model{w: 120, h: 35, phase: phaseReady, st: demoState()}},
		{"terminal too short", model{w: 120, h: 22, phase: phaseReady, st: demoState(), recent: demoRecent(4)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.m.recentVisible() {
				t.Fatal("grid should not be visible in this case")
			}
			next, _ := tc.m.updateKeys(tea.KeyMsg{Type: tea.KeyTab})
			if got := next.(model).focus; got != focusPlayer {
				t.Fatalf("tab from queue landed on %d, want focusPlayer (%d)", got, focusPlayer)
			}
		})
	}
}

// ←/→ step one tile, ↑/↓ jump a whole row, and both clamp at the ends.
func TestRecentGridNavigationClamps(t *testing.T) {
	m := recentModel()
	m.focus = focusRecent

	step := func(k tea.KeyType) {
		next, _ := m.updateKeys(tea.KeyMsg{Type: k})
		m = next.(model)
	}

	step(tea.KeyLeft)
	if m.librarySelection != 0 {
		t.Fatalf("left at the start moved to %d, want 0", m.librarySelection)
	}
	step(tea.KeyRight)
	if m.librarySelection != 1 {
		t.Fatalf("right gave %d, want 1", m.librarySelection)
	}
	step(tea.KeyDown)
	if m.librarySelection != 2 {
		t.Fatalf("down gave %d, want 2", m.librarySelection)
	}
	step(tea.KeyUp)
	if m.librarySelection != 1 {
		t.Fatalf("up gave %d, want 1", m.librarySelection)
	}
	for i := 0; i < 20; i++ {
		step(tea.KeyDown)
	}
	if m.librarySelection != 5 {
		t.Fatalf("down past the end gave %d, want last library item (5)", m.librarySelection)
	}
}

// ↑/↓ must still drive volume when the player is focused, not the grid.
func TestRecentGridDoesNotStealPlayerKeys(t *testing.T) {
	m := recentModel()
	m.focus = focusPlayer
	m.eng = &engine.Engine{}
	m.st.Volume = 50

	next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyUp})
	if got := next.(model); got.st.Volume != 55 || got.recentSel != 0 {
		t.Fatalf("up with player focused: volume=%d sel=%d, want 55 and 0",
			got.st.Volume, got.recentSel)
	}
}

// The transport highlights volume only for the player. Focusing the grid must
// not light it up just because recent now sits at the index player used to have.
func TestVolumeHighlightFollowsPlayerOnly(t *testing.T) {
	// Styles are no-ops without a color profile, which would make every focus
	// render identically.
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.TrueColor)

	volSegs := (demoState().Volume + 9) / 17
	lit := lipgloss.NewStyle().Foreground(accentHi).
		Render(strings.Repeat("▮", volSegs) + strings.Repeat("▯", 6-volSegs))

	for _, tc := range []struct {
		focus int
		want  bool
	}{
		{focusQueue, false},
		{focusRecent, false},
		{focusPlayer, true},
	} {
		m := recentModel()
		m.focus = tc.focus
		if got := strings.Contains(m.transportPanel(m.w-2), lit); got != tc.want {
			t.Errorf("focus %d: volume highlighted = %v, want %v", tc.focus, got, tc.want)
		}
	}
}

// A catalog search returns no Recent; it must not blank the grid.
func TestCatalogSearchKeepsRecentGrid(t *testing.T) {
	m := recentModel()
	next, _ := m.Update(searchMsg{res: engine.SearchResults{
		Songs: []engine.Track{{ID: "x", Title: "Something"}},
	}})
	if got := len(next.(model).recent); got != len(m.recent) {
		t.Fatalf("catalog search left %d recent entries, want %d", got, len(m.recent))
	}
}

// The grid renders exactly the height it is given and never overruns its width.
func TestRecentPanelFitsItsBox(t *testing.T) {
	for _, w := range []int{28, 40, 56} {
		for _, h := range []int{7, 10, 12} {
			m := recentModel()
			m.focus = focusRecent
			m.recentSel = 3
			body := m.recentPanel(w, h)
			lines := strings.Split(body, "\n")
			if len(lines) != h-1 {
				t.Errorf("recentPanel(%d,%d) drew %d lines, want %d", w, h, len(lines), h-1)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw != w {
					t.Errorf("recentPanel(%d,%d) line %d is %d wide, want %d", w, h, i, lw, w)
				}
			}
		}
	}
}

// Enter on a library item opens the matching paged library view.
func TestEnterOnLibraryCategoryOpensAlbumList(t *testing.T) {
	m := recentModel()
	m.focus = focusRecent
	m.librarySelection = 2

	next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if cmd == nil {
		t.Fatal("enter on a library item returned no command")
	}
	if !got.searchOpen || got.sInput {
		t.Fatalf("library overlay state searchOpen=%v input=%v, want opened without text input", got.searchOpen, got.sInput)
	}
	if got.libraryCategory != 1 || got.libraryOffset != 0 || !got.sBusy {
		t.Errorf("library category=%d offset=%d busy=%v, want Albums (1), first page, loading", got.libraryCategory, got.libraryOffset, got.sBusy)
	}
}
