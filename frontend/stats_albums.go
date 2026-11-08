package main

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type statsAlbumMsg struct {
	play    bool
	request uint64
	album   engine.Track
	err     error
}

func (m *model) cancelStatsAlbum() {
	m.statsAlbumRequestID++
	if m.statsAlbumBusy {
		m.statsErr = ""
	}
	m.statsAlbumBusy = false
}
func (h *listeningHistory) albumSongID(period statsRange, now time.Time, name, artist string) string {
	if h == nil {
		return ""
	}
	records := map[string]listeningRecord{}
	start := statsRangeStart(period, now)
	for day, entries := range h.Days {
		date, err := time.ParseInLocation("2006-01-02", day, now.Location())
		if err != nil || (!start.IsZero() && date.Before(start)) {
			continue
		}
		for _, entry := range entries {
			if entry.Album == name && entry.Artist == artist && entry.ID != "" {
				mergeListenRecord(records, entry.ID, entry)
			}
		}
	}
	tracks := topListenRecords(records)
	if len(tracks) == 0 {
		return ""
	}
	return tracks[0].ID
}
func (m model) playStatsAlbum() (tea.Model, tea.Cmd) { return m.loadStatsAlbum(true) }
func (m model) openStatsAlbum() (tea.Model, tea.Cmd) { return m.loadStatsAlbum(false) }
func (m model) loadStatsAlbum(play bool) (tea.Model, tea.Cmd) {
	if m.statsAlbumBusy {
		return m, nil
	}
	albums := m.listenHistory.summary(m.statsRange, time.Now()).Albums
	if len(albums) == 0 {
		return m, nil
	}
	if play && m.audioInitializing() {
		m.statsErr = audioInitializingText
		return m, nil
	}
	selected := albums[max(0, min(m.statsAlbumSel, len(albums)-1))]
	source := m.listenHistory.albumSongID(m.statsRange, time.Now(), selected.Title, selected.Artist)
	if source == "" || m.eng == nil {
		m.statsErr = "This history album cannot be played."
		return m, nil
	}
	m.statsAlbumRequestID++
	request, eng := m.statsAlbumRequestID, m.eng
	m.statsAlbumBusy = true
	m.statsErr = "Loading album…"
	return m, func() tea.Msg {
		res, err := eng.SongAlbum(source)
		var album engine.Track
		if err == nil {
			if len(res.Albums) != 1 || res.Albums[0].ID == "" || res.Albums[0].Kind != "album" {
				err = errors.New("Album information is unavailable for this history entry.")
			} else {
				album = res.Albums[0]
			}
		}
		return statsAlbumMsg{request: request, play: play, album: album, err: err}
	}
}
func (m model) applyStatsAlbum(msg statsAlbumMsg) (tea.Model, tea.Cmd) {
	if !m.statsOpen || m.artistBrowser.open || !m.statsAlbumBusy || msg.request != m.statsAlbumRequestID {
		return m, nil
	}
	m.statsAlbumBusy = false
	m.statsErr = ""
	if msg.err != nil {
		m.statsErr = msg.err.Error()
		return m, nil
	}
	if m.eng == nil {
		m.statsErr = "Connect to Apple Music to play this album."
		return m, nil
	}
	if !msg.play {
		m.artistBrowser = artistBrowserState{open: true, origin: artistFromStatsAlbum}
		return m.openArtistAlbum(msg.album)
	}
	if m.audioInitializing() {
		m.statsErr = audioInitializingText
		return m, nil
	}
	m.statsOpen = false
	return m.playDiscoveryTrack(msg.album)
}
func (m model) updateStatsAlbumMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.statsLayout()
	albums := m.listenHistory.summary(m.statsRange, time.Now()).Albums
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	m.statsFocus = 2
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		m.statsAlbumSel = max(0, min(m.statsAlbumSel+step, len(albums)-1))
		m.statsAlbumOffset = statsListStart(m.statsAlbumSel, m.statsAlbumOffset, l.listRows, len(albums))
		m.statsClickID = ""
	case tea.MouseButtonLeft:
		start := statsListStart(m.statsAlbumSel, m.statsAlbumOffset, l.listRows, len(albums))
		index := start + msg.Y - l.tracksY
		if index < 0 || index >= len(albums) {
			m.statsClickID = ""
			return m, nil
		}
		m.statsAlbumSel, m.statsAlbumOffset = index, start
		m.statsClickID = ""
		return m.openStatsAlbum()
	}
	return m, nil
}
