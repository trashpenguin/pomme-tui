package main

import (
	"encoding/json"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStatsTabReachesAlbumsAndCyclesBack(t *testing.T) {
	m := interactiveStatsModel(120, 38)
	for _, want := range []int{1, 2, 0} {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(model)
		if m.statsFocus != want {
			t.Fatalf("Tab focus=%d want %d", m.statsFocus, want)
		}
	}
}
func TestStatsAlbumPlaysRealAlbumResolvedFromHistorySong(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'e'}}, {Type: tea.KeyRunes, Runes: []rune{'E'}}} {
		t.Run(key.String(), func(t *testing.T) {
			eng, capture, requests := artistProtocolFixture(t)
			m := interactiveStatsModel(120, 38)
			m.eng = eng
			m.statsFocus = 2
			next, lookup := m.Update(key)
			m = next.(model)
			if lookup == nil || !m.statsOpen {
				t.Fatal("album should resolve before playback")
			}
			next, play := m.Update(lookup())
			m = next.(model)
			if play == nil || m.statsOpen || m.loading != "Resolved Album" {
				t.Fatal("resolved album did not start playback")
			}
			if msg := play(); msg != nil {
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
			if params.Kind != "album" || params.ID != "resolved-album" {
				t.Fatalf("wrong album request: %s", raw)
			}
			assertArtistRequest(t, requests, "song_album", "1")
		})
	}
}
func TestStatsAlbumLookupCannotPlayAfterLeavingOrChangingRange(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyRight}, {Type: tea.KeyTab}, {Type: tea.KeyDown}} {
		t.Run(key.String(), func(t *testing.T) {
			eng, _, _ := artistProtocolFixture(t)
			m := interactiveStatsModel(120, 38)
			m.eng = eng
			m.statsFocus = 2
			next, lookup := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if lookup == nil {
				t.Fatal("no album lookup")
			}
			reply := lookup()
			next, _ = m.Update(key)
			m = next.(model)
			_, play := m.Update(reply)
			if play != nil {
				t.Fatal("late lookup started playback after navigation")
			}
		})
	}
}
func TestStatsCompactAlbumsSelectScrollAndMouse(t *testing.T) {
	m := interactiveStatsModel(70, 22)
	m.statsFocus = 2
	day := time.Now().Format("2006-01-02")
	for id, record := range m.listenHistory.Days[day] {
		record.Album = "A" + id
		record.Artist = ""
		m.listenHistory.Days[day][id] = record
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(model)
	if !strings.Contains(m.View(), "› 5.") || !strings.Contains(m.View(), "A5") {
		t.Fatal("fifth album not selected/visible")
	}
	l := m.statsLayout()
	x := l.x + l.widths[0] + l.widths[1] + 7
	next, cmd := m.Update(tea.MouseMsg{X: x, Y: l.tracksY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = next.(model)
	if cmd == nil || m.statsFocus != 2 {
		t.Fatal("click did not select/open album")
	}
	if !strings.Contains(strings.Split(m.View(), "\n")[l.tracksY], "A4") {
		t.Fatal("album scrolled away under mouse")
	}
	next, lookup := m.Update(tea.MouseMsg{X: x, Y: l.tracksY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if lookup == nil || !next.(model).statsOpen {
		t.Fatal("repeated click did not resolve album")
	}
}

func TestStatsAlbumFailureCanRetryAndInitializationBlocksLatePlay(t *testing.T) {
	eng, _, _ := artistProtocolFixture(t)
	m := interactiveStatsModel(120, 38)
	m.eng = eng
	m.statsFocus = 2
	next, lookup := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = next.(model)
	reply := lookup().(statsAlbumMsg)
	reply.err = errors.New("catalog unavailable")
	next, play := m.Update(reply)
	m = next.(model)
	if play != nil || m.statsAlbumBusy || !m.statsOpen || m.statsErr != "catalog unavailable" {
		t.Fatal("lookup failure did not leave a retryable Stats page")
	}
	next, lookup = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = next.(model)
	if lookup == nil {
		t.Fatal("failed lookup could not retry")
	}
	m.beginAudioInitialization()
	next, play = m.Update(lookup())
	if play != nil || !next.(model).statsOpen {
		t.Fatal("late lookup bypassed audio initialization guard")
	}
}
func TestStatsAlbumEmptyAndOfflineDoNotPlay(t *testing.T) {
	for _, empty := range []bool{true, false} {
		m := interactiveStatsModel(120, 38)
		m.statsFocus = 2
		if empty {
			m.listenHistory = nil
		} else {
			m.eng = nil
		}
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil || !next.(model).statsOpen {
			t.Fatal("empty/offline Stats attempted playback")
		}
	}
}

func TestStatsAlbumClickAndEnterOpenTracksWithoutPlaying(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		t.Run(fmt.Sprint(mouse), func(t *testing.T) {
			eng, capture, _ := artistProtocolFixture(t)
			m := interactiveStatsModel(120, 38)
			m.eng = eng
			m.statsFocus = 2
			m.statsRange = statsAllTime
			var event tea.Msg = tea.KeyMsg{Type: tea.KeyEnter}
			if mouse {
				l := m.statsLayout()
				event = tea.MouseMsg{X: l.x + l.widths[0] + l.widths[1] + 7, Y: l.tracksY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
			}
			next, lookup := m.Update(event)
			m = next.(model)
			if lookup == nil {
				t.Fatal("album did not start metadata lookup")
			}
			next, tracks := m.Update(lookup())
			m = next.(model)
			if tracks == nil || !m.artistBrowser.open || m.audioInitializing() {
				t.Fatal("album opened playback instead of track browser")
			}
			next, _ = m.Update(tracks())
			m = next.(model)
			if len(m.artistBrowser.albumSongs.items) != 1 || !strings.Contains(m.View(), "Album Track") {
				t.Fatal("album songs not displayed")
			}
			if _, err := os.Stat(capture); !os.IsNotExist(err) {
				t.Fatal("opening album sent playback request")
			}
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = next.(model)
			if m.artistBrowser.open || !m.statsOpen || m.statsFocus != 2 || m.statsRange != statsAllTime {
				t.Fatal("Back did not restore Stats selection and range")
			}
		})
	}
}
