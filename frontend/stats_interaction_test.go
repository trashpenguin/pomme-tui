package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/muesli/termenv"
)

func interactiveStatsModel(w, h int) model {
	m := mouseModel(w, h, 0)
	m.eng = &engine.Engine{}
	m.statsOpen = true
	entries := map[string]listeningRecord{}
	for i := 1; i <= 5; i++ {
		id := fmt.Sprint(i)
		entries[id] = listeningRecord{ID: id, Title: "No" + id, Artist: "Artist", Album: "Album", Plays: 6 - i, Seconds: 60}
	}
	m.listenHistory = &listeningHistory{Days: map[string]map[string]listeningRecord{time.Now().Format("2006-01-02"): entries}}
	return m
}

func TestStatsSelectAndPlayRankedTrack(t *testing.T) {
	m := interactiveStatsModel(120, 38)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got, cmd := next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	player := got.(model)
	if cmd == nil || player.loading != "No2" || player.statsOpen || !player.audioInitializing() {
		t.Fatalf("selected track was not played: loading=%q open=%v cmd=%v", player.loading, player.statsOpen, cmd != nil)
	}
}

func TestStatsCompactScrollReachesFifthTrack(t *testing.T) {
	m := interactiveStatsModel(70, 22)
	for i := 0; i < 8; i++ {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(model)
	}
	if !strings.Contains(m.View(), "› 5.") {
		t.Fatal("selected fifth track is not visible in compact stats")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || next.(model).loading != "No5" {
		t.Fatal("compact selection did not clamp/play fifth track")
	}
}

func TestStatsBoxesHaveConsistentWidths(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{70, 22}, {100, 30}, {120, 38}, {140, 45}} {
		m := interactiveStatsModel(size[0], size[1])
		found := false
		left, right := -1, -1
		for _, line := range strings.Split(m.View(), "\n") {
			if !strings.Contains(line, "╭") {
				continue
			}
			var starts, ends []int
			for i, r := range line {
				if r == '╭' {
					starts = append(starts, lipgloss.Width(line[:i]))
				}
				if r == '╮' {
					ends = append(ends, lipgloss.Width(line[:i]))
				}
			}
			if len(starts) == 1 {
				if left < 0 {
					left, right = starts[0], ends[0]
				}
				if starts[0] != left || ends[0] != right {
					t.Fatalf("full-width boxes misaligned at %v", size)
				}
			}
			if len(starts) == 3 {
				found = true
				minW, maxW := 1000, 0
				for i := range starts {
					width := ends[i] - starts[i] + 1
					minW = min(minW, width)
					maxW = max(maxW, width)
				}
				if maxW-minW > 1 || starts[0] != left || ends[2] != right {
					t.Fatalf("ranking boxes differ/misalign at %v: starts=%v ends=%v", size, starts, ends)
				}
			}
		}
		if !found {
			t.Fatalf("ranking box row missing at %v", size)
		}
	}
}

func TestStatsLongNamesKeepCountsAndFit(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{70, 22}, {120, 38}, {140, 45}} {
		m := interactiveStatsModel(size[0], size[1])
		day := time.Now().Format("2006-01-02")
		m.listenHistory.Days[day]["1"] = listeningRecord{ID: "1", Title: strings.Repeat("長い曲名", 30), Artist: strings.Repeat("Artist ", 20), Album: "Album", Plays: 99, Seconds: 300}
		view := m.View()
		if !strings.Contains(view, "99 plays") {
			t.Fatal("long track name hid its play count")
		}
		lines := strings.Split(view, "\n")
		if len(lines) > size[1] {
			t.Fatalf("long name overflows height at %v", size)
		}
		for _, line := range lines {
			if lipgloss.Width(line) > size[0] {
				t.Fatalf("long name overflows width at %v", size)
			}
		}
	}
}

func TestStatsRankingTiesHaveStableOrder(t *testing.T) {
	for i := 0; i < 30; i++ {
		got := topListenRecords(map[string]listeningRecord{"b": {ID: "b", Title: "Same", Plays: 1, Seconds: 60}, "a": {ID: "a", Title: "Same", Plays: 1, Seconds: 60}})
		if got[0].ID != "a" || got[1].ID != "b" {
			t.Fatal("tied rankings can change between render and playback")
		}
	}
}

func TestStatsMouseSelectsRenderedTrackAndDoubleClickPlays(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{70, 22}, {140, 45}} {
		m := interactiveStatsModel(size[0], size[1])
		x, y := -1, -1
		for row, line := range strings.Split(m.View(), "\n") {
			if i := strings.Index(line, "No2"); i >= 0 {
				x, y = lipgloss.Width(line[:i]), row
				break
			}
		}
		if x < 0 {
			t.Fatal("second track not rendered")
		}
		event := tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
		next, cmd := m.updateMouse(event)
		m = next.(model)
		if cmd != nil || !strings.Contains(m.View(), "› 2.") {
			t.Fatal("click did not select the rendered second track")
		}
		next, cmd = m.updateMouse(event)
		if cmd == nil || next.(model).loading != "No2" {
			t.Fatal("double-click did not play the selected track")
		}
	}
}

func TestStatsEmptyAndInitializingDoNotStartPlayback(t *testing.T) {
	for _, initializing := range []bool{false, true} {
		m := interactiveStatsModel(100, 30)
		if initializing {
			m.beginAudioInitialization()
		} else {
			m.listenHistory = nil
		}
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil || !next.(model).statsOpen {
			t.Fatal("empty/busy stats attempted playback")
		}
	}
}

func TestStatsDoubleClickAfterScrollingKeepsClickedRowStable(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := interactiveStatsModel(70, 22)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(model)
	x, y := -1, -1
	for row, line := range strings.Split(m.View(), "\n") {
		if i := strings.Index(line, "No4"); i >= 0 {
			x, y = lipgloss.Width(line[:i]), row
			break
		}
	}
	if x < 0 {
		t.Fatal("fourth track not visible after scrolling")
	}
	click := tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	next, _ = m.updateMouse(click)
	m = next.(model)
	if !strings.Contains(strings.Split(m.View(), "\n")[y], "No4") {
		t.Fatal("click shifted the track out from under the mouse")
	}
	next, cmd := m.updateMouse(click)
	if cmd == nil || next.(model).loading != "No4" {
		t.Fatal("double-click after scrolling did not play the clicked track")
	}
}
