package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestLibraryPanelArrowKeysMoveAndClampSelection(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.recent = []engine.Track{{Title: "recent"}}
	m.focus = focusRecent

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
	if m.librarySelection != 5 {
		t.Fatalf("selection moved to %d, want last library item (5)", m.librarySelection)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := updated.(model).librarySelection; got != 4 {
		t.Fatalf("up moved selection to %d, want 4", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := updated.(model).librarySelection; got != 5 {
		t.Fatalf("down moved selection to %d, want 5", got)
	}
}

func TestEnterOpensSelectedLibraryCategory(t *testing.T) {
	for _, tc := range []struct {
		selection int
		category  int
	}{
		{selection: 0, category: 0}, // Discovery opens Songs.
		{selection: 2, category: 1}, // Albums.
		{selection: 5, category: 4}, // Favorites.
	} {
		m := mouseModel(100, 34, 5)
		m.focus, m.librarySelection, m.libraryOffset = focusRecent, tc.selection, 50
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		got := updated.(model)
		if !got.searchOpen || got.sInput || !got.sBusy {
			t.Errorf("selection %d did not open library view: searchOpen=%v input=%v busy=%v", tc.selection, got.searchOpen, got.sInput, got.sBusy)
		}
		if got.libraryCategory != tc.category || got.libraryOffset != 0 {
			t.Errorf("selection %d opened category %d at offset %d, want category %d at offset 0", tc.selection, got.libraryCategory, got.libraryOffset, tc.category)
		}
		if cmd == nil {
			t.Errorf("selection %d did not start loading its library page", tc.selection)
		}
	}
}

func TestClickLibraryItemSelectsAndOpensIt(t *testing.T) {
	m := mouseModel(100, 34, 5)
	m.recent = []engine.Track{{Title: "recent"}}
	lay := m.layout()

	updated, cmd := m.updateMouse(tea.MouseMsg{
		X: 4, Y: lay.recentTop() + 4, // Artists is the fifth row (index 4).
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	got := updated.(model)
	if cmd == nil || !got.searchOpen || got.libraryCategory != 3 || got.librarySelection != 4 {
		t.Fatalf("click opened=%v category=%d selection=%d command=%v, want Artists (3), row 4, and a load command", got.searchOpen, got.libraryCategory, got.librarySelection, cmd != nil)
	}
}
