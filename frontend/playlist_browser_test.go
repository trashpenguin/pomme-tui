package main

import (
	"encoding/json"
	"image/color"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/muesli/termenv"
)

func TestLibraryPagesDoNotCycleCategories(t *testing.T) {
	for _, category := range []int{0, 1, 2, 4} {
		m := mouseModel(120, 38, 0)
		m.libraryOpen, m.libraryPaged = true, true
		m.libraryCategory, m.libraryOffset, m.sSel = category, 50, 3
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		got := next.(model)
		if cmd != nil || got.libraryCategory != category || got.libraryOffset != 50 || got.sSel != 3 || got.sBusy {
			t.Fatalf("Tab changed category %d or its selection/page", category)
		}
	}
	m := artistFixture(120, 38)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil || !next.(model).artistBrowser.open || next.(model).libraryCategory != 3 {
		t.Fatal("Tab left Library Artists")
	}
}

func TestLibraryCollectionsUseCoverGallery(t *testing.T) {
	for _, collection := range []struct {
		selection, category int
		kind, title         string
	}{{2, 1, "album", "ALBUMS"}, {3, 2, "playlist", "PLAYLISTS"}} {
		t.Run(collection.kind, func(t *testing.T) {
			defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
			lipgloss.SetColorProfile(termenv.Ascii)
			for _, size := range [][2]int{{70, 22}, {100, 30}, {140, 45}} {
				m := mouseModel(size[0], size[1], 0)
				m.eng = &engine.Engine{}
				next, cmd := m.openLibrarySelection(collection.selection)
				m = next.(model)
				if !m.artistBrowser.open || cmd == nil {
					t.Fatal("Collection did not open cover gallery")
				}
				next, _ = m.Update(artistBrowserMsg{request: m.artistRequestID, kind: artistLoadList, results: collectionGalleryResults(collection.kind)})
				m = next.(model)
				m.artistArt.put(artistImageKey(m.artistBrowser.root.items[0]), solidImage(4, 4, color.RGBA{R: 200, A: 255}))
				view := m.View()
				if !strings.Contains(view, "POMME · "+collection.title) || !strings.Contains(view, "▀") {
					t.Fatal("missing collection title or terminal cover art")
				}
				if len(strings.Split(view, "\n")) > size[1] {
					t.Fatal("gallery exceeds terminal height")
				}
				for _, line := range strings.Split(view, "\n") {
					if lipgloss.Width(line) > size[0] {
						t.Fatal("gallery exceeds terminal width")
					}
				}
				next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyTab})
				if cmd != nil || !next.(model).artistBrowser.open || next.(model).libraryCategory != collection.category {
					t.Fatal("Tab left collection gallery")
				}
			}

		})
	}
}

func TestCollectionCoverOpensTracksAndPlaysNativeSong(t *testing.T) {
	for _, collection := range []struct {
		selection, category int
		kind, id, title     string
	}{{2, 1, "album", "album-fixture", "ALBUM"}, {3, 2, "playlist", "p.fixture", "PLAYLIST"}} {
		t.Run(collection.kind, func(t *testing.T) {
			eng, capture, requests := artistProtocolFixture(t)
			m := mouseModel(120, 38, 0)
			m.eng = eng
			next, cmd := m.openLibrarySelection(collection.selection)
			m = next.(model)
			next, _ = m.Update(cmd())
			m = next.(model)
			if len(m.artistBrowser.root.items) != 1 {
				t.Fatal("collection response not loaded")
			}
			l := m.artistGeometry()
			next, cmd = m.Update(tea.MouseMsg{X: l.x + 2, Y: l.gridY + 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			m = next.(model)
			if cmd == nil || m.artistBrowser.album.ID != collection.id {
				t.Fatal("collection cover did not open tracks")
			}
			next, _ = m.Update(cmd())
			m = next.(model)
			if !strings.Contains(m.View(), "POMME · "+collection.title) || len(m.artistBrowser.albumSongs.items) != 1 {
				t.Fatal("missing collection songs")
			}
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
			m = next.(model)
			if m.artistBrowser.mode != artistAlbum || m.libraryCategory != collection.category {
				t.Fatal("Tab left collection tracks")
			}
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = next.(model)
			if m.artistBrowser.mode != artistList || !m.artistBrowser.open {
				t.Fatal("Back did not restore collection gallery")
			}
			next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			next, _ = m.Update(cmd())
			m = next.(model)
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatal("missing playback action")
			}
			if msg := cmd(); msg != nil {
				t.Fatalf("play failed: %v", msg)
			}
			raw, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			var params struct{ Kind, ID string }
			if err = json.Unmarshal(raw, &params); err != nil {
				t.Fatal(err)
			}
			if params.Kind != "song" || params.ID != "album-song" {
				t.Fatalf("wrong song played: %s", raw)
			}
			assertArtistRequest(t, requests, "discovery_item", collection.id)
			raw, err = os.ReadFile(requests)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), `"kind":"`+collection.kind+`"`) || !strings.Contains(string(raw), `"category":"`+collection.kind+`s"`) {
				t.Fatal("collection lookup did not use correct category/kind")
			}

		})
	}
}

func collectionGalleryResults(kind string) engine.SearchResults {
	items := []engine.Track{{ID: "collection", Title: strings.Repeat("Collection 長い ", 15), Kind: kind, Art: "cover"}}
	if kind == "album" {
		return engine.SearchResults{Albums: items}
	}
	return engine.SearchResults{Playlists: items}
}
