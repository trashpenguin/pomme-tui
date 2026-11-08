package main

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSearchArtistOpensBrowserAndBackRestoresResults(t *testing.T) {
	eng, capture, _ := artistProtocolFixture(t)
	m := mouseModel(120, 38, 0)
	m.eng = eng
	m.searchOpen = true
	m.sQuery = "Artist"
	m.sTab = 4
	m.sRes.Artists = []engine.Track{{ID: "artist-1", Title: "Artist One", Kind: "artist"}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil || !m.artistBrowser.open || m.searchOpen {
		t.Fatal("artist result did not open shared browser")
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if len(m.artistBrowser.songs.items) != 1 || len(m.artistBrowser.albums.items) != 1 {
		t.Fatal("artist catalogue not visible")
	}
	if _, err := os.Stat(capture); !os.IsNotExist(err) {
		t.Fatal("opening artist played audio")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.artistBrowser.open || !m.searchOpen || m.sQuery != "Artist" || m.sTab != 4 || len(m.sRes.Artists) != 1 {
		t.Fatal("Back lost artist search results")
	}
}
func TestSearchEmptyQueryDoesNotReloadLibrary(t *testing.T) {
	m := mouseModel(100, 30, 0)
	m.searchOpen, m.sInput = true, true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("blank search fetched library instead of prompting for a query")
	}
}
func TestSearchResultsFitAndDistinguishOpenFromPlay(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 30}, {140, 45}} {
		m := mouseModel(size[0], size[1], 0)
		m.searchOpen = true
		m.sTab = 4
		m.sQuery = "artist"
		m.sRes.Artists = []engine.Track{{ID: "artist-1", Title: strings.Repeat("Long Artist 名 ", 20), Kind: "artist"}}
		view := m.View()
		if !strings.Contains(view, "open") {
			t.Fatal("artist result has no open hint")
		}
		if len(strings.Split(view, "\n")) > size[1] {
			t.Fatal("search too tall")
		}
		for _, line := range strings.Split(view, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Fatal("search result overflow")
			}
		}
	}
}

func TestSearchRejectsOldResponsesAndKeepsPartialResults(t *testing.T) {
	m := mouseModel(120, 38, 0)
	m.searchOpen, m.sInput, m.sBusy = true, true, true
	m.sQuery = "old"
	m.searchRequestID = 1
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = next.(model)
	next, _ = m.Update(catalogSearchMsg{request: 1, query: "old", res: engine.SearchResults{Artists: []engine.Track{{ID: "old"}}}})
	m = next.(model)
	if m.sBusy || len(m.sRes.Artists) != 0 {
		t.Fatal("editing query accepted old result or left search stuck")
	}
	next, _ = m.Update(catalogSearchMsg{request: m.searchRequestID, query: m.sQuery, res: engine.SearchResults{Artists: []engine.Track{{ID: "new", Title: "Artist", Kind: "artist"}}}, err: errors.New("Some catalog results are unavailable")})
	m = next.(model)
	if len(m.sRes.Artists) != 1 || m.sErr == "" {
		t.Fatal("partial error discarded valid artist results")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	next, _ = m.Update(catalogSearchMsg{request: m.searchRequestID - 1, query: m.sQuery, res: engine.SearchResults{Artists: []engine.Track{{ID: "late"}}}})
	if next.(model).sRes.Artists[0].ID != "new" {
		t.Fatal("closed search accepted stale result")
	}
}
func TestSearchMouseTabsAndRowsMatchRenderedGeometry(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {140, 45}} {
		m := mouseModel(size[0], size[1], 0)
		m.searchOpen = true
		m.sTab = 4
		m.sRes.Artists = []engine.Track{{ID: "1", Title: "Artist One", Kind: "artist"}, {ID: "2", Title: "Artist Two", Kind: "artist"}}
		l := m.searchGeometry()
		next, _ := m.Update(tea.MouseMsg{X: l.x + 3, Y: l.listY + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(model)
		if m.sSel != 1 || m.sInput {
			t.Fatal("mouse did not select displayed second result")
		}
		next, _ = m.Update(tea.MouseMsg{X: l.x + 2, Y: l.tabY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		if next.(model).sTab != 0 {
			t.Fatal("mouse tab click did not select Recent")
		}
	}
}
func TestSearchCollectionsBrowseWithoutPlayingAndReturnToSearch(t *testing.T) {
	for _, kind := range []string{"album", "playlist"} {
		t.Run(kind, func(t *testing.T) {
			eng, capture, _ := artistProtocolFixture(t)
			m := mouseModel(120, 38, 0)
			m.eng = eng
			m.searchOpen = true
			m.sQuery = "collection"
			item := engine.Track{ID: kind + "-id", Title: "Collection", Kind: kind}
			if kind == "album" {
				m.sTab = 2
				m.sRes.Albums = []engine.Track{item}
			} else {
				m.sTab = 3
				m.sRes.Playlists = []engine.Track{item}
			}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if cmd == nil || !m.artistBrowser.open {
				t.Fatal("collection did not open track page")
			}
			next, _ = m.Update(cmd())
			m = next.(model)
			if _, err := os.Stat(capture); !os.IsNotExist(err) {
				t.Fatal("collection browse played audio")
			}
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = next.(model)
			if m.artistBrowser.open || !m.searchOpen || m.sQuery != "collection" {
				t.Fatal("Back lost collection search")
			}
		})
	}
}
func TestSearchBlankLoadingAndLongResultsFitEverySize(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 30}, {140, 45}} {
		for _, state := range []string{"blank", "busy", "error", "long"} {
			t.Run(fmt.Sprint(size, state), func(t *testing.T) {
				m := mouseModel(size[0], size[1], 0)
				m.searchOpen = true
				m.sInput = true
				switch state {
				case "busy":
					m.sBusy = true
				case "error":
					m.sErr = strings.Repeat("Unavailable ", 20)
				case "long":
					m.sTab = 1
					for i := 0; i < 30; i++ {
						m.sRes.Songs = append(m.sRes.Songs, engine.Track{ID: fmt.Sprint(i), Title: strings.Repeat("長いSong ", 25), Artist: strings.Repeat("Artist ", 20), Kind: "song", Duration: time.Minute})
					}
					m.sSel = 29
					m.sInput = false
				}
				lines := strings.Split(m.View(), "\n")
				if len(lines) > size[1] {
					t.Fatal("search exceeds height")
				}
				for _, line := range lines {
					if lipgloss.Width(line) > size[0] {
						t.Fatal("search exceeds width")
					}
				}
			})
		}
	}
}

func TestSearchScrolledClickKeepsRowStableAndBackRestoresViewport(t *testing.T) {
	eng, _, _ := artistProtocolFixture(t)
	m := mouseModel(120, 38, 0)
	m.eng = eng
	m.searchOpen = true
	m.sTab = 4
	for i := 0; i < 50; i++ {
		m.sRes.Artists = append(m.sRes.Artists, engine.Track{ID: fmt.Sprint(i), Title: fmt.Sprintf("Item%02d", i), Kind: "artist"})
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(model)
	l := m.searchGeometry()
	first := 50 - l.rows
	expected := fmt.Sprintf("Item%02d", first)
	event := tea.MouseMsg{X: l.x + 3, Y: l.listY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	next, _ = m.Update(event)
	m = next.(model)
	if !strings.Contains(strings.Split(m.View(), "\n")[l.listY], expected) {
		t.Fatal("clicked row moved away under pointer")
	}
	next, cmd := m.Update(event)
	m = next.(model)
	if cmd == nil || m.artistBrowser.profile.ID != fmt.Sprint(first) {
		t.Fatal("double click opened wrong artist after scrolling")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if !m.searchOpen || !strings.Contains(strings.Split(m.View(), "\n")[l.listY], expected) {
		t.Fatal("Back lost Search viewport")
	}
}

func TestEmptySearchResultsCanRetryWithEnter(t *testing.T) {
	m := mouseModel(120, 38, 0)
	m.searchOpen = true
	m.sQuery = "retry"
	m.sTerm = "retry"
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !next.(model).sBusy {
		t.Fatal("Enter could not retry an empty search")
	}
}
