package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestLibraryPanelArrowKeysMoveAndClampSelection(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.recent = []engine.Track{{Title: "recent"}}
	m.focus = focusLibrary

	keys := []tea.KeyMsg{
		{Type: tea.KeyDown}, {Type: tea.KeyRight}, {Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyDown},
	}
	var updated tea.Model
	for _, key := range keys {
		var cmd tea.Cmd
		updated, cmd = m.Update(key)
		_ = cmd
		m = updated.(model)
	}
	if m.librarySelection != 6 {
		t.Fatalf("selection moved to %d, want Stats (6)", m.librarySelection)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := updated.(model).librarySelection; got != 5 {
		t.Fatalf("up moved selection to %d, want 5", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := updated.(model).librarySelection; got != 6 {
		t.Fatalf("down moved selection to %d, want 6", got)
	}
}

func TestEnterOpensSelectedLibraryCategory(t *testing.T) {
	for _, tc := range []struct {
		selection int
		category  int
	}{
		{selection: 2, category: 1}, // Albums.
		{selection: 5, category: 4}, // Favorites.
	} {
		m := mouseModel(100, 34, 5)
		m.focus, m.librarySelection, m.libraryOffset = focusLibrary, tc.selection, 50
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		got := updated.(model)
		if !got.libraryOpen || got.searchOpen || got.sInput || !got.sBusy {
			t.Errorf("selection %d did not open library view: libraryOpen=%v searchOpen=%v input=%v busy=%v", tc.selection, got.libraryOpen, got.searchOpen, got.sInput, got.sBusy)
		}
		if got.libraryCategory != tc.category || got.libraryOffset != 0 {
			t.Errorf("selection %d opened category %d at offset %d, want category %d at offset 0", tc.selection, got.libraryCategory, got.libraryOffset, tc.category)
		}
		if cmd == nil {
			t.Errorf("selection %d did not start loading its library page", tc.selection)
		}
	}
}

func TestEnterLibraryDiscoveryAndStatsOpenDedicatedPages(t *testing.T) {
	for _, selection := range []int{0, 6} {
		m := mouseModel(100, 34, 5)
		m.focus, m.librarySelection = focusLibrary, selection
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		got := updated.(model)
		if cmd == nil || got.searchOpen {
			t.Fatalf("selection %d opened search=%v command=%v; want a dedicated page load", selection, got.searchOpen, cmd != nil)
		}
		if selection == 0 && (!got.discoveryOpen || !got.discoveryBusy) {
			t.Fatalf("Discovery open=%v busy=%v, want a loading discovery page", got.discoveryOpen, got.discoveryBusy)
		}
		if selection == 6 && (!got.statsOpen || !got.statsBusy) {
			t.Fatalf("Stats open=%v busy=%v, want a loading stats page", got.statsOpen, got.statsBusy)
		}
	}
}

func TestUppercaseLOpensSelectedDedicatedLibraryPage(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.focus, m.librarySelection = focusLibrary, 3 // Playlists.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	got := updated.(model)
	if cmd == nil || !got.libraryOpen || got.searchOpen || got.libraryCategory != 2 {
		t.Fatalf("L opened library=%v search=%v category=%d cmd=%v; want dedicated Playlists page", got.libraryOpen, got.searchOpen, got.libraryCategory, cmd != nil)
	}
}

func TestUppercaseLOpensDedicatedPageFromSearchResults(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.searchOpen, m.sInput, m.libraryCategory = true, false, 2 // Playlists.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	got := updated.(model)
	if cmd == nil || !got.libraryOpen || got.searchOpen || got.libraryCategory != 2 {
		t.Fatalf("L from search opened library=%v search=%v category=%d cmd=%v; want dedicated Playlists page", got.libraryOpen, got.searchOpen, got.libraryCategory, cmd != nil)
	}
}

func TestCtrlLDoesNothingInSearchOrLibrary(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.searchOpen, m.libraryOpen, m.libraryPaged = true, true, true
	m.libraryCategory, m.libraryOffset, m.sBusy = 2, 50, false

	searchUpdated, searchCmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyCtrlL})
	search := searchUpdated.(model)
	if searchCmd != nil || search.libraryCategory != 2 || search.libraryOffset != 50 || search.sBusy {
		t.Fatalf("Ctrl+L changed search/library state: category=%d offset=%d busy=%v command=%v", search.libraryCategory, search.libraryOffset, search.sBusy, searchCmd != nil)
	}

	libraryUpdated, libraryCmd := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyCtrlL})
	library := libraryUpdated.(model)
	if libraryCmd != nil || library.libraryCategory != 2 || library.libraryOffset != 50 || library.sBusy {
		t.Fatalf("Ctrl+L changed library state: category=%d offset=%d busy=%v command=%v", library.libraryCategory, library.libraryOffset, library.sBusy, libraryCmd != nil)
	}
}

func TestClickLibraryItemSelectsAndOpensIt(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.recent = []engine.Track{{Title: "recent"}}
	lay := m.layout()

	updated, cmd := m.updateMouse(tea.MouseMsg{
		X: 4, Y: lay.libraryTop() + 4, // Artists is the fifth row (index 4).
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	got := updated.(model)
	if cmd == nil || !got.libraryOpen || got.searchOpen || got.libraryCategory != 3 || got.librarySelection != 4 {
		t.Fatalf("click library=%v search=%v category=%d selection=%d command=%v, want Artists (3), row 4, and a load command", got.libraryOpen, got.searchOpen, got.libraryCategory, got.librarySelection, cmd != nil)
	}
}

func TestLibraryCategoriesUseDedicatedPageAndSlashOpensSearch(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.focus, m.librarySelection = focusLibrary, 1
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if cmd == nil || !got.libraryOpen || got.searchOpen {
		t.Fatalf("Songs opened library=%v search=%v command=%v, want dedicated library page", got.libraryOpen, got.searchOpen, cmd != nil)
	}
	if view := got.View(); strings.Contains(view, "SEARCH APPLE MUSIC") || !strings.Contains(view, "POMME · LIBRARY") {
		t.Fatalf("library view should be distinct from search:\n%s", view)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if next := updated.(model); !next.searchOpen || !next.sInput || next.libraryOpen {
		t.Fatalf("slash did not hand off to search: library=%v search=%v input=%v", next.libraryOpen, next.searchOpen, next.sInput)
	}
}

func TestClickDiscoveryOpensDedicatedPage(t *testing.T) {
	m := mouseModel(100, 34, 5)
	lay := m.layout()
	updated, cmd := m.updateMouse(tea.MouseMsg{
		X: 4, Y: lay.libraryTop(),
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	got := updated.(model)
	if cmd == nil || !got.discoveryOpen || got.searchOpen || got.librarySelection != 0 {
		t.Fatalf("click opened discovery=%v search=%v selection=%d cmd=%v; want dedicated Discovery page", got.discoveryOpen, got.searchOpen, got.librarySelection, cmd != nil)
	}
}

func TestDiscoveryViewShowsAppleShelvesAndOpensPlaylistTracks(t *testing.T) {
	m := mouseModel(120, 38, 5)
	m.discoveryOpen = true
	m.discoveryFeed = engine.Discovery{Sections: []engine.DiscoverySection{{
		Title: "Made For Us",
		Items: []engine.Track{{ID: "mix", Title: "J-Pop Mix", Kind: "playlist", Artist: "Apple Music"}},
	}}}
	if view := m.View(); !strings.Contains(view, "Made For Us") || !strings.Contains(view, "J-Pop Mix") || !strings.Contains(view, "w Save (confirm)") || !strings.Contains(view, "a Queue") {
		t.Fatalf("Discovery view is missing its recommendation shelf or controls:\n%s", view)
	}
	updated, cmd := m.updateDiscovery(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !updated.(model).discoveryDetail {
		t.Fatal("Enter did not open the selected playlist's track page")
	}
	updated, _ = updated.(model).Update(discoveryTracksMsg{tracks: []engine.Track{{Title: "First Track", Kind: "song"}}})
	if view := updated.(model).View(); !strings.Contains(view, "First Track") || !strings.Contains(view, "P Queue whole list") {
		t.Fatal("opened discovery playlist did not render its tracks")
	}
	back, _ := updated.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := back.(model); !got.discoveryOpen || got.discoveryDetail {
		t.Fatalf("Escape from playlist tracks should return to Discover, got open=%v detail=%v", got.discoveryOpen, got.discoveryDetail)
	}
}

func TestDiscoveryQueueAndSaveControls(t *testing.T) {
	m := mouseModel(120, 38, 5)
	m.eng = &engine.Engine{}
	m.discoveryOpen = true
	m.discoveryFeed = engine.Discovery{Sections: []engine.DiscoverySection{{Items: []engine.Track{{ID: "mix", Title: "J-Pop Mix", Kind: "playlist"}}}}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	got := updated.(model)
	if cmd == nil || got.note != "adding to queue: J-Pop Mix" {
		t.Fatalf("a did not queue the playlist: cmd=%v note=%q", cmd != nil, got.note)
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd == nil || updated.(model).note != "adding to next: J-Pop Mix" {
		t.Fatalf("A did not queue playlist next: cmd=%v note=%q", cmd != nil, updated.(model).note)
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	got = updated.(model)
	if cmd != nil || !got.saveConfirmOpen || got.saveConfirmTrack.Title != "J-Pop Mix" {
		t.Fatalf("w did not ask before saving the playlist: cmd=%v confirm=%v track=%q", cmd != nil, got.saveConfirmOpen, got.saveConfirmTrack.Title)
	}
	if view := got.View(); !strings.Contains(view, "SAVE TO LIBRARY?") || !strings.Contains(view, "y / Enter") || !strings.Contains(view, "n / Esc") {
		t.Fatalf("save confirmation does not explain its keys:\n%s", view)
	}
	updated, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil || updated.(model).saveConfirmOpen {
		t.Fatalf("n did not cancel library save: cmd=%v confirm=%v", cmd != nil, updated.(model).saveConfirmOpen)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	updated, cmd = updated.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil || updated.(model).saveConfirmOpen || updated.(model).note != "saving to library: J-Pop Mix" {
		t.Fatalf("y did not confirm library save: cmd=%v confirm=%v note=%q", cmd != nil, updated.(model).saveConfirmOpen, updated.(model).note)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if got := updated.(model); !got.queueOpen || got.focus != focusQueue {
		t.Fatalf("q did not open queue manager: open=%v focus=%d", got.queueOpen, got.focus)
	}
}

func TestQueueManagerRemovalAndReorderingControls(t *testing.T) {
	m := mouseModel(120, 38, 5)
	m.eng = &engine.Engine{}
	m.queueOpen = true
	m.st.Queue = []engine.Track{{ID: "a", Title: "First"}, {ID: "b", Title: "Second"}}
	m.selIdx = 0
	view := m.View()
	for _, text := range []string{"POMME · QUEUE MANAGER", "del remove", "ctrl+↑/↓ reorder"} {
		if !strings.Contains(view, text) {
			t.Errorf("queue manager is missing %q:\n%s", text, view)
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if cmd == nil || updated.(model).selIdx != 1 {
		t.Fatalf("ctrl+down did not move queue selection: sel=%d cmd=%v", updated.(model).selIdx, cmd != nil)
	}
	updated, cmd = updated.(model).Update(tea.KeyMsg{Type: tea.KeyDelete})
	if cmd == nil || !strings.Contains(updated.(model).note, "removed from queue: Second") {
		t.Fatalf("Delete did not remove selected queue item: cmd=%v note=%q", cmd != nil, updated.(model).note)
	}
}

func TestStatsViewShowsLibraryAndPlaybackCounts(t *testing.T) {
	m := mouseModel(120, 38, 5)
	m.statsOpen = true
	m.listenHistory = &listeningHistory{Days: map[string]map[string]listeningRecord{
		time.Now().Format("2006-01-02"): {
			"song": {ID: "song", Title: "Blah Blah", Artist: "Joji", Album: "Piss In The Wind", Plays: 2, Seconds: 240},
		},
	}}
	view := m.View()
	for _, text := range []string{"POMME · STATS", "Last 7 Days", "Last 30 Days", "This Month", "This Year", "All Time", "TOP TRACKS", "TOP ARTISTS", "TOP ALBUMS", "Blah Blah", "Joji", "Piss In The Wind", "LAST 10 DAYS", "2 plays"} {
		if !strings.Contains(view, text) {
			t.Errorf("Stats view is missing %q:\n%s", text, view)
		}
	}
}

func TestStatsViewFitsSmallAndWideTerminals(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 30}, {140, 45}} {
		m := mouseModel(size[0], size[1], 5)
		m.statsOpen = true
		lines := strings.Split(m.View(), "\n")
		if len(lines) > size[1] {
			t.Errorf("Stats view at %dx%d has %d rows", size[0], size[1], len(lines))
		}
		for i, line := range lines {
			if width := lipgloss.Width(line); width > size[0] {
				t.Errorf("Stats view at %dx%d row %d is %d columns", size[0], size[1], i, width)
			}
		}
	}
}
