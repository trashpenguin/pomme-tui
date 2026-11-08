package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func artistProtocolFixture(t *testing.T) (*engine.Engine, string, string) {
	t.Helper()
	folder := t.TempDir()
	fixtures := filepath.Join(folder, "fixtures.json")
	capture := filepath.Join(folder, "play.json")
	requests := filepath.Join(folder, "requests.jsonl")
	data := `{"library_page":{"albums":[{"id":"album-fixture","title":"Album One","kind":"album"}],"playlists":[{"id":"p.fixture","title":"Playlist One","kind":"playlist"}],"artists":[{"id":"artist-1","title":"Artist One","kind":"artist"}]},"song_album":{"albums":[{"id":"resolved-album","title":"Resolved Album","kind":"album"}]},"song_artists":{"artists":[{"id":"artist-1","title":"Artist","kind":"artist"}]},"artist_page":{"artists":[{"id":"artist-1","title":"Artist One","kind":"artist"}],"songs":[{"id":"top-song","title":"Top Song","kind":"song"}],"albums":[{"id":"album-1","title":"Album One","kind":"album"}]},"discovery_item":[{"id":"album-song","title":"Album Track","kind":"song"}]}`
	if err := os.WriteFile(fixtures, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AMTUI_NATIVE_BACKEND", exe)
	t.Setenv("POMME_STATS_TEST_BACKEND", "1")
	t.Setenv("POMME_STATS_PLAY_CAPTURE", capture)
	t.Setenv("POMME_ARTIST_TEST_FIXTURES", fixtures)
	t.Setenv("POMME_ARTIST_RPC_CAPTURE", requests)
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)
	return eng, capture, requests
}

func TestArtistLibraryAlbumSongFlowUsesNativeProtocol(t *testing.T) {
	eng, capture, requests := artistProtocolFixture(t)
	m := mouseModel(120, 38, 0)
	m.eng = eng
	next, cmd := m.openLibrarySelection(4)
	m = next.(model)
	next, _ = m.Update(cmd())
	m = next.(model)
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	next, _ = m.Update(cmd())
	m = next.(model)
	if m.artistBrowser.profile.ID != "artist-1" || len(m.artistBrowser.albums.items) != 1 {
		t.Fatal("artist profile/catalogue response did not reach the browser")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	next, _ = m.Update(cmd())
	m = next.(model)
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("no song playback command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("play request failed: %v", msg)
	}
	raw, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var params struct{ Kind, ID string }
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	if params.Kind != "song" || params.ID != "album-song" {
		t.Fatalf("wrong playback request: %s", raw)
	}
	assertArtistRequest(t, requests, "artist_page", "artist-1")
	assertArtistRequest(t, requests, "discovery_item", "album-1")
}

func TestStatsArtistResolvesSongCreditsBeforeOpeningProfile(t *testing.T) {
	eng, _, requests := artistProtocolFixture(t)
	m := interactiveStatsModel(120, 38)
	m.eng = eng
	m, lookup := m.ensureStatsArtist("Artist", false)
	next, _ := m.Update(lookup())
	m = next.(model)
	next, cmd := m.openStatsArtist("Artist", false)
	m = next.(model)
	if cmd == nil {
		t.Fatal("resolved artist did not request profile")
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if m.artistBrowser.profile.ID != "artist-1" {
		t.Fatal("artist lookup opened the wrong profile")
	}
	assertArtistRequest(t, requests, "song_artists", "1")
	assertArtistRequest(t, requests, "artist_page", "artist-1")
}

func assertArtistRequest(t *testing.T, path, method, id string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		var request struct {
			Method string
			Params struct{ ID string }
		}
		if json.Unmarshal(scan.Bytes(), &request) != nil {
			t.Fatal("invalid fixture request")
		}
		if request.Method == method && request.Params.ID == id {
			return
		}
	}
	t.Fatalf("did not send %s with real catalog ID %s", method, id)
}

func TestArtistQueueSuccessKeepsSongsPlayable(t *testing.T) {
	for _, key := range []rune{'a', 'A'} {
		t.Run(string(key), func(t *testing.T) {
			eng, _, _ := artistProtocolFixture(t)
			m := artistFixture(120, 38)
			m.eng = eng
			m.artistBrowser.mode = artistDetail
			m.artistBrowser.profile = m.artistBrowser.root.items[0]
			m.artistBrowser.songs.items = []engine.Track{{ID: "top-song", Title: "Top Song", Kind: "song"}}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
			m = next.(model)
			if cmd == nil {
				t.Fatal("no queue action")
			}
			next, _ = m.Update(cmd())
			m = next.(model)
			if m.artistBrowser.err != "" || !strings.Contains(m.View(), "Top Song") {
				t.Fatal("successful queue action replaced the artist page with an error")
			}
			_, play := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if play == nil {
				t.Fatal("queue success blocked playback")
			}
		})
	}
}

func TestStatsArtistChooserRetryUsesSongCredits(t *testing.T) {
	eng, _, requests := artistProtocolFixture(t)
	m := interactiveStatsModel(120, 38)
	m.eng = eng
	m.artistBrowser = artistBrowserState{open: true, origin: artistFromStats, mode: artistList, lookupName: "Artist", root: artistGridState{items: []engine.Track{{ID: "one", Title: "Same Name", Kind: "artist"}, {ID: "two", Title: "Same Name", Kind: "artist"}}}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = next.(model)
	if cmd == nil {
		t.Fatal("retry did not issue an artist lookup")
	}
	_, _ = m.Update(cmd())
	assertArtistRequest(t, requests, "song_artists", "1")
}
