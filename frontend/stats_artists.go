package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type statsArtistResult struct {
	request  uint64
	sourceID string
	artists  []engine.Track
	loading  bool
	err      string
}
type statsArtistMsg struct {
	request uint64
	key     string
	artists []engine.Track
	err     error
}

func (h *listeningHistory) artistTracks(period statsRange, now time.Time, name string) []listeningRecord {
	records := map[string]listeningRecord{}
	if h == nil {
		return nil
	}
	start := statsRangeStart(period, now)
	for day, entries := range h.Days {
		date, err := time.ParseInLocation("2006-01-02", day, now.Location())
		if err != nil || (!start.IsZero() && date.Before(start)) {
			continue
		}
		for _, entry := range entries {
			if entry.Artist == name && entry.ID != "" {
				mergeListenRecord(records, entry.ID, entry)
			}
		}
	}
	return topListenRecords(records)
}

func (m model) statsArtistKey(name string) (string, string) {
	key := fmt.Sprintf("%d\x00%s", m.statsRange, name)
	if cached, ok := m.statsArtistData[key]; ok {
		return key, cached.sourceID
	}
	tracks := m.listenHistory.artistTracks(m.statsRange, time.Now(), name)
	id := ""
	if len(tracks) > 0 {
		id = tracks[0].ID
	}
	return key, id
}

func statsArtistProfile(items []engine.Track, name string) (engine.Track, bool) {
	var matched engine.Track
	matches := 0
	for _, item := range items {
		if strings.EqualFold(artistLabel(item.Title), artistLabel(name)) {
			matched = item
			matches++
		}
	}
	if matches == 1 {
		return matched, true
	}
	if matches > 1 {
		return engine.Track{}, false
	}
	if len(items) == 1 {
		return items[0], true
	}
	return engine.Track{}, false
}

func (m model) ensureStatsArtist(name string, force bool) (model, tea.Cmd) {
	key, source := m.statsArtistKey(name)
	if m.statsArtistData == nil || len(m.statsArtistData) > 64 {
		m.statsArtistData = map[string]statsArtistResult{}
	}
	if _, ok := m.statsArtistData[key]; ok && !force {
		return m, nil
	}
	if source == "" || m.eng == nil {
		m.statsArtistData[key] = statsArtistResult{err: "Artist information is unavailable for this history entry.", sourceID: source}
		return m, nil
	}
	m.statsArtistRequestID++
	request := m.statsArtistRequestID
	m.statsArtistData[key] = statsArtistResult{loading: true, request: request, sourceID: source}
	eng := m.eng
	return m, func() tea.Msg {
		res, err := eng.SongArtists(source)
		return statsArtistMsg{key: key, request: request, artists: res.Artists, err: err}
	}
}

func statsArtistCardRows(l statsLayout) int {
	if l.compact {
		return 2
	}
	return 3
}
func statsArtistVisible(l statsLayout) int { return max(1, (l.panelHeight-1)/statsArtistCardRows(l)) }

func (m model) requestStatsArtists() (model, tea.Cmd) {
	if !m.statsOpen || m.artistBrowser.open {
		return m, nil
	}
	l := m.statsLayout()
	records := m.listenHistory.summary(m.statsRange, time.Now()).Artists
	start := statsListStart(m.statsArtistSel, m.statsArtistOffset, statsArtistVisible(l), len(records))
	var cmds []tea.Cmd
	var pictures []engine.Track
	for i := start; i < min(start+statsArtistVisible(l), len(records)); i++ {
		var cmd tea.Cmd
		m, cmd = m.ensureStatsArtist(records[i].Title, false)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		key, _ := m.statsArtistKey(records[i].Title)
		if artist, ok := statsArtistProfile(m.statsArtistData[key].artists, records[i].Title); ok {
			pictures = append(pictures, artist)
		}
	}
	var imageCmd tea.Cmd
	m, imageCmd = m.requestArtistImages(pictures)
	if imageCmd != nil {
		cmds = append(cmds, imageCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) presentStatsArtist(result statsArtistResult) (tea.Model, tea.Cmd) {
	if result.err != "" {
		m.artistBrowser.busy = false
		m.artistBrowser.err = result.err
		return m, nil
	}
	if profile, ok := statsArtistProfile(result.artists, m.artistBrowser.lookupName); ok {
		return m.openArtistProfile(profile)
	}
	m.artistBrowser.mode, m.artistBrowser.busy = artistList, false
	m.artistBrowser.root = artistGridState{items: result.artists}
	if len(result.artists) == 0 {
		m.artistBrowser.err = "No catalog artists were found for this history entry."
	}
	return m.requestVisibleArtistImages()
}

func (m model) openStatsArtist(name string, force bool) (tea.Model, tea.Cmd) {
	key, _ := m.statsArtistKey(name)
	if !m.artistBrowser.open {
		m.artistBrowser = artistBrowserState{open: true, origin: artistFromStats}
	}
	m.artistRequestID++
	m.artistBrowser.mode, m.artistBrowser.busy, m.artistBrowser.err = artistResolving, true, ""
	m.artistBrowser.lookupKey, m.artistBrowser.lookupName = key, name
	if existing, ok := m.statsArtistData[key]; ok && !force {
		if existing.loading {
			return m, nil
		}
		if existing.err == "" {
			return m.presentStatsArtist(existing)
		}
	}
	var cmd tea.Cmd
	m, cmd = m.ensureStatsArtist(name, force)
	if result := m.statsArtistData[key]; !result.loading {
		return m.presentStatsArtist(result)
	}
	return m, cmd
}

func (m model) applyStatsArtist(msg statsArtistMsg) (tea.Model, tea.Cmd) {
	current, ok := m.statsArtistData[msg.key]
	if !ok || current.request != msg.request {
		return m, nil
	}
	result := statsArtistResult{artists: msg.artists, request: msg.request, sourceID: current.sourceID}
	if msg.err != nil {
		result.err = msg.err.Error()
	}
	m.statsArtistData[msg.key] = result
	if m.artistBrowser.open && m.artistBrowser.mode == artistResolving && m.artistBrowser.lookupKey == msg.key {
		return m.presentStatsArtist(result)
	}
	return m.requestStatsArtists()
}

func (m model) statsArtistCards(records []listeningRecord, l statsLayout) string {
	width := l.widths[1]
	cardRows := statsArtistCardRows(l)
	photoCols := 2 * cardRows
	textWidth := max(1, width-photoCols-2)
	start := statsListStart(m.statsArtistSel, m.statsArtistOffset, statsArtistVisible(l), len(records))
	selected := max(0, min(m.statsArtistSel, len(records)-1))
	var lines []string
	for i := start; i < min(start+statsArtistVisible(l), len(records)); i++ {
		record := records[i]
		key, _ := m.statsArtistKey(record.Title)
		profile, ok := statsArtistProfile(m.statsArtistData[key].artists, record.Title)
		if !ok {
			profile = engine.Track{Title: record.Title, Kind: "artist"}
		}
		photo := m.artistPicture(profile, photoCols, cardRows)
		prefix := fmt.Sprintf("%d. ", i+1)
		style := lipgloss.NewStyle().Foreground(fgDim)
		if m.statsFocus == 1 && i == selected {
			prefix = "› " + prefix
			style = style.Foreground(accentHi).Bold(true)
		}
		unit := "plays"
		if record.Plays == 1 {
			unit = "play"
		}
		captions := []string{prefix + artistLabel(record.Title), fmt.Sprintf("%d %s · %d/%d", record.Plays, unit, i+1, len(records)), "↵ open artist"}
		for row := 0; row < cardRows; row++ {
			lines = append(lines, pad(photo[row]+"  "+style.Render(ansi.Truncate(captions[row], textWidth, "…")), width))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(fgFaint).Render(" no plays yet"))
	}
	for len(lines) < l.panelHeight-1 {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m model) updateStatsArtistMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.statsLayout()
	records := m.listenHistory.summary(m.statsRange, time.Now()).Artists
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	m.statsFocus, m.statsClickID = 1, ""
	visible := statsArtistVisible(l)
	if msg.Button == tea.MouseButtonWheelDown || msg.Button == tea.MouseButtonWheelUp {
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		m.statsArtistSel = max(0, min(m.statsArtistSel+step, len(records)-1))
		m.statsArtistOffset = statsListStart(m.statsArtistSel, m.statsArtistOffset, visible, len(records))
		return m.requestStatsArtists()
	}
	if msg.Button == tea.MouseButtonLeft {
		row := msg.Y - l.tracksY
		if row >= visible*statsArtistCardRows(l) {
			return m, nil
		}
		start := statsListStart(m.statsArtistSel, m.statsArtistOffset, visible, len(records))
		index := start + row/statsArtistCardRows(l)
		if index >= len(records) {
			return m, nil
		}
		m.statsArtistSel, m.statsArtistOffset = index, start
		return m.openStatsArtist(records[index].Title, false)
	}
	return m, nil
}
