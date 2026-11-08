package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"strings"
	"testing"
	"time"
)

func TestQueueManagerHonorsInitializationAndCtrlC(t *testing.T) {
	m := mouseModel(100, 34, 2)
	m.queueOpen = true
	m.beginAudioInitialization()
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd != nil || !m.audioInitializing() {
		t.Fatal("queue Enter bypassed initialization guard")
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil || next.(model).queueOpen {
		t.Fatal("Ctrl+C did not return from queue manager")
	}
}
func TestMainLibraryBrowsingWorksWhileAudioInitializes(t *testing.T) {
	m := mouseModel(100, 34, 0)
	m.focus = focusLibrary
	m.librarySelection = 6
	m.beginAudioInitialization()
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !next.(model).statsOpen {
		t.Fatal("audio initialization blocked safe Library navigation")
	}
}
func TestMainQueueClearRequiresQueueFocus(t *testing.T) {
	for _, focus := range []int{focusLibrary, focusRecent, focusPlayer} {
		m := mouseModel(100, 34, 2)
		m.focus = focus
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
		if cmd != nil {
			t.Fatal("queue clear fired from another focused panel")
		}
	}
}
func TestSearchLibraryShortcutMatchesSearchCategory(t *testing.T) {
	m := mouseModel(100, 34, 0)
	m.eng = &engine.Engine{}
	m.searchOpen = true
	m.sTab = 4
	m.libraryCategory = 0
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	if next.(model).libraryCategory != 3 {
		t.Fatal("Search L opened stale Library category instead of Artists")
	}
}
func TestQueueAndSaveNoticesDoNotFailPendingPlayback(t *testing.T) {
	for _, save := range []bool{false, true} {
		t.Run(map[bool]string{false: "queue", true: "save"}[save], func(t *testing.T) {
			eng, _, _ := artistProtocolFixture(t)
			m := mouseModel(100, 34, 1)
			m.eng = eng
			m.beginAudioInitialization()
			var cmd tea.Cmd
			if save {
				m.saveQueueUntil = time.Now().Add(time.Second)
				next, action := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
				m = next.(model)
				cmd = action
			} else {
				cmd = m.queueDiscoveryCmd(engine.Track{ID: "song-1", Title: "Song", Kind: "song"}, false)
			}
			if cmd == nil {
				t.Fatal("no queue/save action")
			}
			next, _ := m.Update(cmd())
			m = next.(model)
			if m.initFailed || !m.initPending {
				t.Fatal("non-playback notice falsely failed initialization")
			}
		})
	}
}
func TestF1HelpOpensFromEveryBrowseScreen(t *testing.T) {
	for _, screen := range []string{"search", "library", "artist", "stats", "discovery", "queue", "account"} {
		t.Run(screen, func(t *testing.T) {
			m := mouseModel(100, 34, 0)
			switch screen {
			case "search":
				m.searchOpen, m.sInput = true, true
			case "library":
				m.libraryOpen = true
			case "artist":
				m.artistBrowser.open = true
			case "stats":
				m.statsOpen = true
			case "discovery":
				m.discoveryOpen = true
			case "queue":
				m.queueOpen = true
			case "account":
				m.acctOpen = true
			}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF1})
			if cmd != nil || !next.(model).helpOpen {
				t.Fatal("F1 did not open help")
			}
		})
	}
}

func TestCtrlXClearsOnlyTheFocusedQueue(t *testing.T) {
	m := mouseModel(100, 34, 2)
	m.focus = focusQueue
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	if cmd == nil {
		t.Fatal("Ctrl+X did not clear focused queue")
	}
}

func TestHelpNeverConsumesQuestionMarkInSearchInput(t *testing.T) {
	m := mouseModel(100, 34, 0)
	m.searchOpen, m.sInput = true, true
	m.sQuery = "why"
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if cmd != nil || next.(model).helpOpen || next.(model).sQuery != "why?" {
		t.Fatal("help stole query text")
	}
}
func TestReverseTabMatchesFocusAndArtistTabs(t *testing.T) {
	m := mouseModel(120, 38, 0)
	m.focus = focusQueue
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if next.(model).focus != focusPlayer {
		t.Fatal("Shift+Tab did not reverse main focus")
	}
	m = artistFixture(120, 38)
	m.artistBrowser.mode = artistDetail
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if next.(model).artistBrowser.tab != 1 {
		t.Fatal("Shift+Tab did not switch artist details")
	}
}
func TestControlActionsNeverLeakOutOfHelp(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.helpOpen = true
	for _, key := range []tea.KeyMsg{{Type: tea.KeySpace}, {Type: tea.KeyCtrlX}, {Type: tea.KeyEnter}, {Type: tea.KeyRunes, Runes: []rune{'r'}}} {
		next, cmd := m.Update(key)
		if cmd != nil || !next.(model).helpOpen || next.(model).st.Playing != m.st.Playing {
			t.Fatal("help triggered underlying action")
		}
	}
}

func TestSearchQueryCanContainSlash(t *testing.T) {
	m := mouseModel(100, 34, 0)
	m.searchOpen, m.sInput = true, true
	m.sQuery = "AC"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if next.(model).sQuery != "AC/" {
		t.Fatal("query editing swallowed literal slash")
	}
	m = next.(model)
	m.sInput = false
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !next.(model).sInput || next.(model).sQuery != "AC/" {
		t.Fatal("slash result shortcut changed query")
	}
}
func TestMiniPlayerNeverHidesActiveBrowseScreens(t *testing.T) {
	for _, screen := range []string{"library", "artist", "stats", "discovery", "queue"} {
		t.Run(screen, func(t *testing.T) {
			m := mouseModel(120, 38, 0)
			m.mini = true
			title := ""
			switch screen {
			case "library":
				m.libraryOpen = true
				title = "POMME · LIBRARY"
			case "artist":
				m.artistBrowser.open = true
				title = "POMME · ARTISTS"
			case "stats":
				m.statsOpen = true
				title = "POMME · STATS"
			case "discovery":
				m.discoveryOpen = true
				title = "POMME · DISCOVERY"
			case "queue":
				m.queueOpen = true
				title = "POMME · QUEUE MANAGER"
			}
			if !strings.Contains(m.View(), title) {
				t.Fatal("mini mode hid active " + screen)
			}
		})
	}
}
