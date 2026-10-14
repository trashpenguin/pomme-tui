package main

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"os"
	"testing"
	"time"
)

func TestNativeLiveUILeavesInitializationAndRendersLyrics(t *testing.T) {
	if os.Getenv("NATIVE_LIVE_TEST") != "1" {
		t.Skip("requires signed-in native backend")
	}
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	m := model{phase: phaseReady, eng: eng, artCache: newArtCache(8), statusCh: make(chan string, 8), w: 100, h: 32}
	m.beginAudioInitialization()
	if err := eng.Play("song", "1535067323"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		st, err := eng.State()
		if err != nil {
			t.Fatal(err)
		}
		next, _ := m.Update(stateMsg{st: st})
		m = next.(model)
		if !m.audioInitializing() && m.st.Playing {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if m.audioInitializing() || !m.st.Playing {
		t.Fatalf("UI remains initializing: native=%+v pending=%t seen=%t", m.st, m.initPending, m.initSeen)
	}
	ly, err := lyrics.Fetch(context.Background(), m.st.Now.Artist, m.st.Now.Title, m.st.Now.Duration)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(lyricsMsg{id: m.st.Now.ID, ly: ly})
	m = next.(model)
	image, err := fetchArtwork(context.Background(), artworkURL(m.st.Now.Art, 256))
	if err != nil {
		t.Fatal("artwork:", err)
	}
	next, _ = m.Update(artMsg{id: m.st.Now.ID, img: image})
	m = next.(model)
	if len(m.ly.Lines) == 0 || m.art == nil {
		t.Fatal("UI lyric/artwork update failed")
	}
	if view := m.View(); len(view) == 0 {
		t.Fatal("empty interface")
	}
	t.Logf("UI ready; lyrics=%d synchronized=%t; artwork=%dx%d", len(m.ly.Lines), m.ly.Synced, m.art.Bounds().Dx(), m.art.Bounds().Dy())
	if err := eng.PlayPause(); err != nil {
		t.Fatal(err)
	}
	if err := eng.SetVolume(35); err != nil {
		t.Fatal(err)
	}
	if err := eng.SeekTo(12 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := eng.Reload(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	st, err := eng.State()
	if err != nil {
		t.Fatal(err)
	}
	next, _ = m.Update(stateMsg{st: st})
	m = next.(model)
	if m.audioInitializing() || m.st.Playing || m.st.Volume != 35 {
		t.Fatal("paused reload blocks UI or changes audio state")
	}
	_, resume := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if resume == nil {
		t.Fatal("Space blocked after paused reload")
	}
	resume()
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if st.Playing {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !st.Playing || st.Volume != 35 || st.Pos < 11*time.Second {
		t.Fatal("paused reload did not resume at retained position and gain")
	}
	t.Log("Paused reload -> Space resumed at retained position/volume")
}
