package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"testing"
)

func TestQueuePagingAndBounds(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyPgDown, tea.KeyCtrlF} {
		m := model{phase: phaseReady, w: 100, h: 30, focus: focusQueue, st: engine.State{Queue: make([]engine.Track, 80)}}
		next, _ := m.Update(tea.KeyMsg{Type: key})
		m = next.(model)
		if m.selIdx != m.layout().queueRows() {
			t.Fatalf("page did not move queue: %d", m.selIdx)
		}
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		m = next.(model)
		if m.selIdx != 0 {
			t.Fatal("page up did not return")
		}
		m.selIdx = 79
		next, _ = m.Update(tea.KeyMsg{Type: key})
		if next.(model).selIdx != 79 {
			t.Fatal("escaped queue bounds")
		}
	}
}
func TestSearchPagingDoesNotLoadLibrary(t *testing.T) {
	m := model{phase: phaseReady, searchOpen: true, h: 30, sTab: 1, sRes: engine.SearchResults{Songs: make([]engine.Track, 30)}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = next.(model)
	if cmd != nil || m.sSel != 12 || m.sBusy {
		t.Fatal("search page unexpectedly requested library")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if next.(model).sSel != 0 {
		t.Fatal("fallback page up failed")
	}
	m.libraryPaged = true
	m.sSel = 49
	m.sRes.Songs = make([]engine.Track, 50)
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if cmd == nil || next.(model).libraryOffset != 50 {
		t.Fatal("library boundary did not fetch next batch")
	}
	m = next.(model)
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if cmd != nil || next.(model).libraryOffset != 50 {
		t.Fatal("paging while busy skipped a batch")
	}
}
func TestHelpPagingAndResize(t *testing.T) {
	m := model{helpOpen: true, w: 70, h: 22}
	before := m.helpView()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = next.(model)
	if !m.helpOpen || m.helpPage != 1 || m.helpView() == before {
		t.Fatal("help page did not change")
	}
	for i := 0; i < 20; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
		m = next.(model)
	}
	if m.helpPage != m.helpPageCount()-1 {
		t.Fatal("help page escaped bounds")
	}
	m.h = 100
	_ = m.helpView()
}
