package main

import (
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func runArtistImageCommand(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			m = runArtistImageCommand(t, m, child)
		}
		return m
	}
	if _, ok := msg.(artistImageMsg); !ok {
		t.Fatalf("unexpected image command message %T", msg)
	}
	next, _ := m.Update(msg)
	return next.(model)
}

func TestArtistPortraitRequestsAreDeduplicatedAndBounded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/96x96bb.jpg" {
			t.Errorf("unexpanded artwork path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_ = png.Encode(w, solidImage(400, 400, color.RGBA{G: 180, A: 255}))
	}))
	defer server.Close()
	item := engine.Track{ID: "artist", Kind: "artist", Title: "Artist", Art: server.URL + "/{w}x{h}bb.jpg"}
	m := mouseModel(100, 30, 0)
	m, cmd := m.requestArtistImages([]engine.Track{item, item})
	if cmd == nil {
		t.Fatal("portrait not requested")
	}
	m, duplicate := m.requestArtistImages([]engine.Track{item})
	if duplicate != nil {
		t.Fatal("duplicate in-flight portrait request")
	}
	m = runArtistImageCommand(t, m, cmd)
	img, ok := m.artistArt.get(artistImageKey(item))
	if !ok || img.Bounds().Dx() > 96 || img.Bounds().Dy() > 96 {
		t.Fatal("portrait cache retained an oversized or missing image")
	}
	_, again := m.requestArtistImages([]engine.Track{item})
	if again != nil || calls.Load() != 1 {
		t.Fatal("cached portrait was downloaded again")
	}
}

func TestMissingPortraitDoesNotBlockArtistNavigation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	m := artistFixture(100, 30)
	m.artistBrowser.root.items[0].Art = server.URL
	m, cmd := m.requestArtistImages(m.artistBrowser.root.items[:1])
	m = runArtistImageCommand(t, m, cmd)
	next, cmd := m.activateArtistItem()
	got := next.(model)
	if cmd == nil || got.artistBrowser.profile.ID != "artist-1" || got.artistBrowser.err != "" {
		t.Fatal("a missing picture prevented artist navigation")
	}
}
