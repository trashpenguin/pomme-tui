package main

import (
	"image"
	"image/color"
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

// Tab must walk queue → library → recent → player and wrap back.
func TestTabCyclesFourZones(t *testing.T) {
	m := recentModel()
	want := []int{focusLibrary, focusRecent, focusPlayer, focusQueue}
	for i, w := range want {
		next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(model)
		if m.focus != w {
			t.Fatalf("tab #%d put focus on %d, want %d", i+1, m.focus, w)
		}
	}
}

// Tab skips the recent list when there are no tracks or too little room.
func TestTabSkipsHiddenRecentList(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    model
	}{
		{"terminal too short", model{w: 120, h: 22, phase: phaseReady, st: demoState(), recent: demoRecent(4)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.m.recentVisible() {
				t.Fatal("recent list should not be visible in this case")
			}
			next, _ := tc.m.updateKeys(tea.KeyMsg{Type: tea.KeyTab})
			if got := next.(model).focus; got != focusLibrary {
				t.Fatalf("tab from queue landed on %d, want focusLibrary (%d)", got, focusLibrary)
			}
			next, _ = next.(model).Update(tea.KeyMsg{Type: tea.KeyTab})
			if got := next.(model).focus; got != focusPlayer {
				t.Fatalf("tab past hidden recent list landed on %d, want focusPlayer (%d)", got, focusPlayer)
			}
		})
	}
}

func TestRecentPanelShowsEmptyState(t *testing.T) {
	m := model{w: 120, h: 35, phase: phaseReady, st: demoState()}
	if !m.recentVisible() {
		t.Fatal("recent panel should remain visible before listening history is available")
	}
	if !strings.Contains(m.View(), "nothing played yet") {
		t.Fatal("empty recent panel did not explain that there are no recent tracks")
	}
}

// Library focus keeps category navigation separate from the recent track list.
func TestLibraryNavigationClamps(t *testing.T) {
	m := recentModel()
	m.focus = focusLibrary

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
	if m.librarySelection != 6 {
		t.Fatalf("down past the end gave %d, want Stats (6)", m.librarySelection)
	}
}

func TestRecentListNavigationClamps(t *testing.T) {
	m := recentModel()
	m.focus = focusRecent
	m.recentSel = 0

	for i := 0; i < len(m.recent)+3; i++ {
		next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(model)
	}
	if m.recentSel != len(m.recent)-1 {
		t.Fatalf("down past the end selected %d, want %d", m.recentSel, len(m.recent)-1)
	}
	next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyUp})
	if got := next.(model).recentSel; got != len(m.recent)-2 {
		t.Fatalf("up selected %d, want %d", got, len(m.recent)-2)
	}
}

// ↑/↓ must still drive volume when the player is focused, not either panel.
func TestLibraryPanelsDoNotStealPlayerKeys(t *testing.T) {
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
		{focusLibrary, false},
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

func TestClickRecentTrackSelectsAndStartsPlayback(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.recent = demoRecent(6)
	m.eng = &engine.Engine{}
	lay := m.layout()

	next, cmd := m.updateMouse(tea.MouseMsg{
		X: 4, Y: lay.recentTop() + recentCardRows + 1,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	got := next.(model)
	if cmd == nil || got.focus != focusRecent || got.recentSel != 1 {
		t.Fatalf("click selected focus=%d track=%d and command=%v; want track 1 and playback", got.focus, got.recentSel, cmd != nil)
	}
	if got.loading != m.recent[1].Title {
		t.Fatalf("click loading %q, want %q", got.loading, m.recent[1].Title)
	}
}

// A catalog search returns no Recent; it must keep the listening history.
func TestCatalogSearchKeepsRecentList(t *testing.T) {
	m := recentModel()
	next, _ := m.Update(searchMsg{res: engine.SearchResults{
		Songs: []engine.Track{{ID: "x", Title: "Something"}},
	}})
	if got := len(next.(model).recent); got != len(m.recent) {
		t.Fatalf("catalog search left %d recent entries, want %d", got, len(m.recent))
	}
}

func TestRecentPanelRendersBelowLibrary(t *testing.T) {
	m := recentModel()
	rows := strings.Split(m.View(), "\n")
	libraryRow, recentRow := -1, -1
	for i, row := range rows {
		if strings.Contains(row, "LIBRARY") {
			libraryRow = i
		}
		if strings.Contains(row, "RECENTLY PLAYED ·") {
			recentRow = i
		}
	}
	if libraryRow < 0 || recentRow < 0 || recentRow <= libraryRow {
		t.Fatalf("panel order is library=%d recently-played=%d; expected both, with recently played below", libraryRow, recentRow)
	}
	if !strings.Contains(strings.Join(rows, "\n"), "Album With A Fairly Long Name") {
		t.Fatal("recently played track names are missing from the panel")
	}
}

// The compact recent list renders exactly the height it is given and fits its width.
func TestRecentListPanelFitsItsBox(t *testing.T) {
	for _, w := range []int{28, 40, 56} {
		for _, h := range []int{4, 6, 8} {
			m := recentModel()
			m.focus = focusRecent
			m.recentSel = 3
			body := m.recentListPanel(w, h)
			lines := strings.Split(body, "\n")
			if len(lines) != h-1 {
				t.Errorf("recentListPanel(%d,%d) drew %d lines, want %d", w, h, len(lines), h-1)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw != w {
					t.Errorf("recentListPanel(%d,%d) line %d is %d wide, want %d", w, h, i, lw, w)
				}
			}
		}
	}
}

func TestRecentListPanelShowsCachedArtwork(t *testing.T) {
	m := recentModel()
	m.focus = focusRecent
	m.artCache = newArtCache(8)
	m.artCache.put(m.recent[0].ID, image.NewUniform(color.RGBA{R: 220, G: 40, B: 90, A: 255}))
	body := m.recentListPanel(40, 6)
	first := strings.Split(body, "\n")[0]
	if !strings.Contains(first, "▀") || strings.Contains(first, "░░░░░░") {
		t.Fatalf("cached album art was not rendered as a thumbnail: %q", first)
	}
	lines := strings.Split(body, "\n")
	for i := 0; i < recentArtworkRows; i++ {
		if !strings.Contains(lines[i], "▀") {
			t.Errorf("artwork row %d is missing: %q", i, lines[i])
		}
	}
}

func TestRecentArtworkRequestsOnlyVisibleTracks(t *testing.T) {
	m := recentModel()
	m.artCache = newArtCache(8)
	updated, cmd := m.requestRecentArtwork()
	if cmd == nil || len(updated.recentArtRequested) != 2 {
		t.Fatalf("first view requested %d tracks with command=%v, want two visible tracks", len(updated.recentArtRequested), cmd != nil)
	}
	updated.recentSel = len(updated.recent) - 1
	updated, cmd = updated.requestRecentArtwork()
	if cmd == nil || len(updated.recentArtRequested) != 4 {
		t.Fatalf("scrolled view requested %d tracks with command=%v, want the two visible tracks plus the first two", len(updated.recentArtRequested), cmd != nil)
	}
}

// Enter on a library item opens the matching paged library view.
func TestEnterOnLibraryCategoryOpensAlbumList(t *testing.T) {
	m := recentModel()
	m.focus = focusLibrary
	m.librarySelection = 2

	next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if cmd == nil {
		t.Fatal("enter on a library item returned no command")
	}
	if !got.libraryOpen || got.searchOpen || got.sInput {
		t.Fatalf("library page state libraryOpen=%v searchOpen=%v input=%v, want dedicated page without text input", got.libraryOpen, got.searchOpen, got.sInput)
	}
	if got.libraryCategory != 1 || got.libraryOffset != 0 || !got.sBusy {
		t.Errorf("library category=%d offset=%d busy=%v, want Albums (1), first page, loading", got.libraryCategory, got.libraryOffset, got.sBusy)
	}
}
