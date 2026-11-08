package main

import (
	"encoding/json"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestEPlaysWholeAlbumFromGalleryArtistAndTracks(t *testing.T) {
	for _, where := range []string{"library", "artist", "tracks"} {
		for _, key := range []rune{'e', 'E'} {
			t.Run(where+string(key), func(t *testing.T) {
				eng, capture, _ := artistProtocolFixture(t)
				m := artistFixture(120, 38)
				m.eng = eng
				album := engine.Track{ID: "whole-album", Title: "Whole Album", Kind: "album"}
				switch where {
				case "library":
					m.artistBrowser.origin = artistFromAlbums
					m.artistBrowser.root.items = []engine.Track{album}
				case "artist":
					m.artistBrowser.mode = artistDetail
					m.artistBrowser.tab = 1
					m.artistBrowser.albums.items = []engine.Track{album}
				case "tracks":
					m.artistBrowser.mode = artistAlbum
					m.artistBrowser.album = album
					m.artistBrowser.albumSongs.items = []engine.Track{{ID: "one-song", Kind: "song"}}
				}
				next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
				if cmd == nil {
					t.Fatal("e did not start album playback")
				}
				got := next.(model)
				if got.artistBrowser.open || got.libraryOpen || !got.audioInitializing() {
					t.Fatal("album play did not enter guarded playback flow")
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
				if params.Kind != "album" || params.ID != album.ID {
					t.Fatalf("did not play whole album: %s", raw)
				}
			})
		}
	}
}

func TestAlbumEHonorsLoadingErrorsAndInitialization(t *testing.T) {
	for _, state := range []string{"busy", "error", "initializing", "offline", "song"} {
		t.Run(state, func(t *testing.T) {
			m := artistFixture(120, 38)
			m.artistBrowser.root.items = []engine.Track{{ID: "album", Kind: "album"}}
			switch state {
			case "busy":
				m.artistBrowser.busy = true
			case "error":
				m.artistBrowser.err = "lookup failed"
			case "initializing":
				m.initPending = true
			case "offline":
				m.eng = nil
			case "song":
				m.artistBrowser.root.items[0].Kind = "song"
			}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			if cmd != nil || !next.(model).artistBrowser.open {
				t.Fatal("e bypassed playback guard or closed browser")
			}
		})
	}
}
