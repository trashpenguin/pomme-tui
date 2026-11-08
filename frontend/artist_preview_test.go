package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/muesli/termenv"
)

// Optional visual QA artifact from local catalogue fixtures, never a live account.
func TestArtistPreviewFrames(t *testing.T) {
	folder := os.Getenv("POMME_ARTIST_PREVIEW_FIXTURE_DIR")
	if folder == "" {
		t.Skip("optional local visual fixture")
	}
	var data struct {
		Artist                      engine.Track
		Page                        engine.SearchResults
		AlbumTracks, LibraryArtists []engine.Track
	}
	raw, err := os.ReadFile(filepath.Join(folder, "artist-catalogue.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	saved := theme{accent: accent, accentHi: accentHi, accentLo: accentLo, fgBright: fgBright, fgDim: fgDim, fgFaint: fgFaint, borderDim: borderDim, selBg: selBg}
	defer applyTheme(saved)
	applyTheme(*themeByName("gruvbox"))
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := artistFixture(140, 45)
	m.artistBrowser.root.items = data.LibraryArtists
	items := append([]engine.Track{data.Artist}, data.LibraryArtists...)
	items = append(items, data.Page.Albums[:min(6, len(data.Page.Albums))]...)
	m, cmd := m.requestArtistImages(items)
	m = runArtistImageCommand(t, m, cmd)
	save := func(name string, frame string) {
		if err := os.WriteFile(filepath.Join(folder, name+".ansi"), []byte(frame), 0600); err != nil {
			t.Fatal(err)
		}
	}
	save("artists-gallery", m.View())
	m.artistBrowser.mode = artistDetail
	m.artistBrowser.profile = data.Artist
	m.artistBrowser.songs.items = data.Page.Songs
	m.artistBrowser.albums.items = data.Page.Albums
	save("artist-songs", m.View())
	m.artistBrowser.tab = 1
	save("artist-albums", m.View())
	stats := interactiveStatsModel(140, 45)
	stats.statsFocus = 1
	stats.artistArt = m.artistArt
	stats.statsArtistData = map[string]statsArtistResult{}
	entries := map[string]listeningRecord{}
	for i, artist := range data.LibraryArtists[:min(2, len(data.LibraryArtists))] {
		id := artist.ID
		entries[id] = listeningRecord{ID: id, Title: "Preview song", Artist: artist.Title, Album: "Sample album", Plays: 10 - i, Seconds: 600}
	}
	stats.listenHistory = &listeningHistory{Days: map[string]map[string]listeningRecord{time.Now().Format("2006-01-02"): entries}}
	for _, artist := range data.LibraryArtists[:min(2, len(data.LibraryArtists))] {
		key, source := stats.statsArtistKey(artist.Title)
		stats.statsArtistData[key] = statsArtistResult{sourceID: source, artists: []engine.Track{artist}}
	}
	save("stats-artists", stats.View())
	search := mouseModel(140, 45, 0)
	search.searchOpen = true
	search.sTab = 4
	search.sQuery = "Modern Baseball"
	search.sTerm = search.sQuery
	search.sRes = data.Page
	search.sRes.Artists = []engine.Track{data.Artist}
	search.artistArt = m.artistArt
	save("search-artists", search.View())
	help := mouseModel(100, 48, 0)
	help.helpOpen = true
	save("controls-wide", help.View())
	help.w, help.h = 70, 22
	save("controls-narrow", help.View())

	mini := mouseModel(140, 14, 0)
	mini.mini = true
	mini.st.Playing = true
	mini.st.Volume = 80
	mini.st.Now = engine.Track{ID: "preview", Title: "Drown (New)", Artist: "Bring Me The Horizon", Album: "That's the Spirit", Kind: "song"}
	mini.st.Pos, mini.st.Dur = 12*time.Second, 222*time.Second
	save("mini-wide", mini.View())
	mini.w, mini.h = 40, 8
	save("mini-narrow", mini.View())

}
