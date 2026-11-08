package main

import (
	"image/color"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/muesli/termenv"
)

func artistFixture(w, h int) model {
	m := mouseModel(w, h, 0)
	m.eng = &engine.Engine{}
	m.artistBrowser = artistBrowserState{open: true, origin: artistFromLibrary, mode: artistList, root: artistGridState{items: []engine.Track{{ID: "artist-1", Title: "Artist One", Kind: "artist"}, {ID: "artist-2", Title: "Artist Two", Kind: "artist"}}}}
	m.libraryOpen = true
	m.libraryCategory = 3
	return m
}

func TestLibraryArtistsOpensPortraitBrowser(t *testing.T) {
	m := mouseModel(120, 38, 0)
	m.eng = &engine.Engine{}
	next, cmd := m.openLibrarySelection(4)
	got := next.(model)
	if cmd == nil || !got.artistBrowser.open || got.artistBrowser.mode != artistList || !got.artistBrowser.busy {
		t.Fatal("Artists did not open a loading portrait browser")
	}
}

func TestArtistPortraitCardsAndDetailFitSupportedSizes(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{70, 22}, {100, 30}, {120, 38}, {140, 45}} {
		m := artistFixture(size[0], size[1])
		m.artistArt = newArtCache(16)
		m.artistBrowser.root.items[0].Art = "portrait"
		m.artistArt.put(artistImageKey(m.artistBrowser.root.items[0]), solidImage(4, 4, color.RGBA{R: 200, A: 255}))
		for _, mode := range []artistMode{artistList, artistDetail, artistAlbum} {
			m.artistBrowser.mode = mode
			m.artistBrowser.profile = m.artistBrowser.root.items[0]
			m.artistBrowser.songs.items = []engine.Track{{ID: "song", Title: strings.Repeat("長い曲名", 25), Kind: "song"}}
			m.artistBrowser.albums.items = []engine.Track{{ID: "album", Title: strings.Repeat("Long Album ", 15), Kind: "album"}}
			m.artistBrowser.album = m.artistBrowser.albums.items[0]
			m.artistBrowser.albumSongs = m.artistBrowser.songs
			view := m.View()
			if mode != artistAlbum && !strings.Contains(view, "▀") {
				t.Fatal("artist portrait did not use the existing artwork renderer")
			}
			lines := strings.Split(view, "\n")
			if len(lines) > size[1] {
				t.Fatalf("artist mode %v exceeds height at %v", mode, size)
			}
			for _, line := range lines {
				if lipgloss.Width(line) > size[0] {
					t.Fatalf("artist mode %v exceeds width at %v", mode, size)
				}
			}
		}
	}
}

func TestClickArtistOpensItsSongsAndAlbums(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{70, 22}, {140, 45}} {
		m := artistFixture(size[0], size[1])
		x, y := -1, -1
		for row, line := range strings.Split(m.View(), "\n") {
			if i := strings.Index(line, "Artist Two"); i >= 0 {
				x, y = lipgloss.Width(line[:i]), row
				break
			}
		}
		if x < 0 {
			t.Fatal("artist card missing")
		}
		next, cmd := m.updateMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(model)
		if cmd == nil || m.artistBrowser.mode != artistDetail || m.artistBrowser.profile.ID != "artist-2" {
			t.Fatal("click did not open the artist under the pointer")
		}
		next, _ = m.Update(artistBrowserMsg{request: m.artistRequestID, kind: artistLoadProfile, results: engine.SearchResults{Artists: []engine.Track{{ID: "artist-2", Title: "Artist Two", Kind: "artist"}}, Songs: []engine.Track{{ID: "song", Title: "Top Song", Kind: "song"}}, Albums: []engine.Track{{ID: "album", Title: "Album One", Kind: "album"}}}})
		m = next.(model)
		if !strings.Contains(m.View(), "Top Song") || !strings.Contains(m.View(), "Albums") {
			t.Fatal("artist detail is missing songs/albums")
		}
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(model)
		if !strings.Contains(m.View(), "Album One") {
			t.Fatal("Albums tab did not show the artist's albums")
		}
		next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
		if cmd == nil || m.artistBrowser.mode != artistAlbum {
			t.Fatal("album did not open its tracks")
		}
		next, _ = m.Update(artistBrowserMsg{request: m.artistRequestID, kind: artistLoadAlbum, tracks: []engine.Track{{ID: "album-song", Title: "Album Track", Kind: "song"}}})
		m = next.(model)
		next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
		if cmd == nil || m.artistBrowser.open || m.libraryOpen || m.loading != "Album Track" {
			t.Fatal("album track did not enter normal playback")
		}
	}
}

func TestArtistBackAndStaleResponsesPreserveOrigin(t *testing.T) {
	m := artistFixture(120, 38)
	m.artistBrowser.root.selected = 1
	next, _ := m.openArtistProfile(m.artistBrowser.root.items[1])
	m = next.(model)
	old := m.artistRequestID
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.artistBrowser.mode != artistList || m.artistBrowser.root.selected != 1 {
		t.Fatal("Back lost artist grid position")
	}
	next, _ = m.Update(artistBrowserMsg{request: old, kind: artistLoadProfile, results: engine.SearchResults{Artists: []engine.Track{{ID: "wrong", Title: "Wrong artist"}}}})
	m = next.(model)
	if m.artistBrowser.mode != artistList || m.artistBrowser.profile.ID == "wrong" {
		t.Fatal("late response replaced the current artist view")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.artistBrowser.open || m.libraryOpen {
		t.Fatal("Back from library artists did not return to player")
	}
}

func TestStatsArtistOpensSharedBrowserAndReturnsToStats(t *testing.T) {
	m := interactiveStatsModel(120, 38)
	m.statsRange = statsAllTime
	next, lookupCmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if (cmd == nil && lookupCmd == nil) || !m.artistBrowser.open || m.artistBrowser.origin != artistFromStats {
		t.Fatal("Stats artist did not open shared browser through real artist lookup")
	}
	next, cmd = m.Update(statsArtistMsg{key: m.artistBrowser.lookupKey, request: m.statsArtistData[m.artistBrowser.lookupKey].request, artists: []engine.Track{{ID: "real-artist-id", Title: "Artist", Kind: "artist"}}})
	m = next.(model)
	if cmd == nil || m.artistBrowser.mode != artistDetail || m.artistBrowser.profile.ID != "real-artist-id" {
		t.Fatal("history song lookup did not open the resolved catalog artist")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.artistBrowser.open || !m.statsOpen || m.statsRange != statsAllTime {
		t.Fatal("artist Back lost the Stats view/range")
	}
}

func TestStatsArtistRetryIgnoresEarlierLookup(t *testing.T) {
	m := interactiveStatsModel(120, 38)
	m, _ = m.ensureStatsArtist("Artist", true)
	key, _ := m.statsArtistKey("Artist")
	old := m.statsArtistData[key].request
	m, _ = m.ensureStatsArtist("Artist", true)
	next, _ := m.Update(statsArtistMsg{key: key, request: old, artists: []engine.Track{{ID: "stale", Title: "Stale", Kind: "artist"}}})
	got := next.(model).statsArtistData[key]
	if !got.loading || len(got.artists) > 0 {
		t.Fatal("an older lookup replaced a retry in progress")
	}
}

func TestArtistEmptyAlbumsExplainsTheEmptyState(t *testing.T) {
	m := artistFixture(100, 30)
	m.artistBrowser.mode = artistDetail
	m.artistBrowser.tab = 1
	m.artistBrowser.profile = m.artistBrowser.root.items[0]
	if !strings.Contains(m.View(), "No albums available") {
		t.Fatal("empty albums tab renders an unexplained blank area")
	}
}

func TestArtistNameCollisionsRequireAChoice(t *testing.T) {
	items := []engine.Track{{ID: "one", Title: "Same Name", Kind: "artist"}, {ID: "two", Title: "Same Name", Kind: "artist"}}
	if _, ok := statsArtistProfile(items, "Same Name"); ok {
		t.Fatal("same-name catalog artists were guessed instead of offering a choice")
	}
}

func TestClosedArtistResponseDoesNotUnlockAnotherLibraryLoad(t *testing.T) {
	m := artistFixture(100, 30)
	m.artistRequestID = 3
	m.artistBrowser.open = false
	m.sBusy = true
	next, _ := m.Update(artistBrowserMsg{request: 2, kind: artistLoadProfile})
	if !next.(model).sBusy || next.(model).artistBrowser.open {
		t.Fatal("stale artist response changed the current library load")
	}
}

func TestReconnectInvalidatesPendingArtistRequests(t *testing.T) {
	m := artistFixture(100, 30)
	m.eng = nil
	m.artistRequestID = 3
	m.artistBrowser.busy = true
	next, _ := m.reconnect("reconnecting")
	m = next.(model)
	if m.artistBrowser.open {
		t.Fatal("an artist page from the old connection remained active")
	}
	next, _ = m.Update(artistBrowserMsg{request: 3, kind: artistLoadProfile, results: engine.SearchResults{Artists: []engine.Track{{ID: "old", Title: "Old"}}}})
	if next.(model).artistBrowser.profile.ID == "old" {
		t.Fatal("old connection response repopulated the artist browser")
	}
}
