package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type statsLayout struct {
	innerWidth            int
	widths                [3]int
	compact               bool
	listRows, panelHeight int
	x, tracksY            int
}

// Widths include exactly three borders and two gaps. Mouse hit testing uses
// the same centered layout and scrolling window as the renderer.
func (m model) statsLayout() statsLayout {
	l := statsLayout{innerWidth: max(20, min(120, m.w-8)-2), compact: m.h < 37, listRows: 5, panelHeight: 8}
	if l.compact {
		l.listRows, l.panelHeight = 2, 3
	}
	available := l.innerWidth - 6
	l.widths = [3]int{(available + 2) / 3, (available + 1) / 3, available / 3}
	height, titleRows := 37, 1
	if l.compact {
		height, titleRows = 22, 0
	}
	l.x = max(0, (m.w-l.innerWidth-2)/2)
	l.tracksY = max(0, (m.h-height)/2) + titleRows + 5 + 5 + 2
	return l
}

func statsListStart(selected, offset, rows, count int) int {
	selected = max(0, min(selected, count-1))
	offset = max(0, min(offset, max(0, count-rows)))
	if selected < offset {
		return selected
	}
	if selected >= offset+rows {
		return selected - rows + 1
	}
	return offset
}

func statsTruncate(text string, width int) string { return ansi.Truncate(text, width, "…") }

func (m model) playStatsTrack() (tea.Model, tea.Cmd) {
	tracks := m.listenHistory.summary(m.statsRange, time.Now()).Tracks
	if len(tracks) == 0 {
		return m, nil
	}
	if m.audioInitializing() {
		m.statsErr = audioInitializingText
		return m, nil
	}
	selected := tracks[max(0, min(m.statsTrackSel, len(tracks)-1))]
	if selected.ID == "" || m.eng == nil {
		m.statsErr = "This history entry cannot be played."
		return m, nil
	}
	m.statsOpen = false
	return m.playDiscoveryTrack(engine.Track{ID: selected.ID, Title: selected.Title, Artist: selected.Artist, Album: selected.Album, Kind: "song"})
}

func (m model) updateStatsMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.statsLayout()
	if msg.Action == tea.MouseActionPress {
		m.cancelStatsAlbum()
	}
	albumX := l.x + l.widths[0] + l.widths[1] + 6
	if msg.X > albumX && msg.X < albumX+l.widths[2]+1 && msg.Y >= l.tracksY && msg.Y < l.tracksY+l.listRows {
		return m.updateStatsAlbumMouse(msg)
	}
	artistX := l.x + l.widths[0] + 3
	if msg.X > artistX && msg.X < artistX+l.widths[1]+1 && msg.Y >= l.tracksY && msg.Y < l.tracksY+l.panelHeight-1 {
		return m.updateStatsArtistMouse(msg)
	}
	tracks := m.listenHistory.summary(m.statsRange, time.Now()).Tracks
	if msg.X <= l.x || msg.X >= l.x+l.widths[0]+1 || msg.Y < l.tracksY || msg.Y >= l.tracksY+l.listRows || msg.Action != tea.MouseActionPress {
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			m.statsClickID = ""
		}
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		m.statsFocus = 0
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		m.statsTrackSel = max(0, min(m.statsTrackSel+step, len(tracks)-1))
		m.statsTrackOffset = statsListStart(m.statsTrackSel, m.statsTrackOffset, l.listRows, len(tracks))
		m.statsClickID = ""
	case tea.MouseButtonLeft:
		m.statsFocus = 0
		start := statsListStart(m.statsTrackSel, m.statsTrackOffset, l.listRows, len(tracks))
		index := start + msg.Y - l.tracksY
		if index >= len(tracks) {
			return m, nil
		}
		m.statsTrackSel = index
		m.statsTrackOffset = start
		id, now := tracks[index].ID, time.Now()
		if id != "" && id == m.statsClickID && now.Sub(m.statsClickAt) < 400*time.Millisecond {
			m.statsClickID = ""
			return m.playStatsTrack()
		}
		m.statsClickID, m.statsClickAt = id, now
	}
	return m, nil
}
