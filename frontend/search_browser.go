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

type catalogSearchMsg struct {
	request uint64
	query   string
	res     engine.SearchResults
	err     error
}
type searchQueueMsg struct {
	request uint64
	notice  string
	err     error
}
type searchLayout struct{ x, y, width, inner, rows, listY, listW, photoRows, photoCols, tabY int }

func (m model) searchGeometry() searchLayout {
	l := searchLayout{width: min(112, m.w-8), rows: m.searchRows()}
	l.inner = l.width - 2
	l.photoRows = min(10, l.rows)
	l.photoCols = 2 * l.photoRows
	l.listW = l.inner - l.photoCols - 3
	l.x = max(0, (m.w-l.width)/2)
	l.y = max(0, (m.h-l.rows-11)/2)
	l.listY = l.y + 7
	l.tabY = l.y + 3
	return l
}
func (m model) selectedSearchItem() engine.Track {
	list := m.curList()
	if m.sSel < 0 || m.sSel >= len(list) {
		return engine.Track{}
	}
	return list[m.sSel]
}
func (m model) requestSearchArtwork() (model, tea.Cmd) {
	if !m.searchOpen || m.sBusy {
		return m, nil
	}
	m.sOffset = statsListStart(m.sSel, m.sOffset, m.searchRows(), len(m.curList()))
	return m.requestArtistImages([]engine.Track{m.selectedSearchItem()})
}
func (m model) applyCatalogSearch(msg catalogSearchMsg) (tea.Model, tea.Cmd) {
	if msg.request != m.searchRequestID || !m.searchOpen || m.artistBrowser.open {
		return m, nil
	}
	m.sBusy = false
	if msg.query != strings.TrimSpace(m.sQuery) {
		return m, nil
	}
	m.sErr, m.sNotice = "", ""
	if msg.err != nil {
		m.sErr = msg.err.Error()
	}
	msg.res.Recent = m.sRes.Recent
	m.sRes, m.sSel, m.sInput = msg.res, 0, false
	m.sOffset = 0
	m.sTerm = msg.query
	m.libraryPaged = false
	if m.sTab == 0 {
		m.sTab = 1
	}
	return m.requestSearchArtwork()
}
func (m model) activateSearchItem(playAll bool) (tea.Model, tea.Cmd) {
	if m.sBusy || m.eng == nil {
		return m, nil
	}
	item := m.selectedSearchItem()
	if item.ID == "" {
		return m, nil
	}
	if !playAll && (item.Kind == "artist" || item.Kind == "album" || item.Kind == "playlist") {
		m.searchRequestID++
		m.searchOpen = false
		m.artistBrowser = artistBrowserState{open: true, origin: artistFromSearch}
		if item.Kind == "artist" {
			return m.openArtistProfile(item)
		}
		return m.openArtistAlbum(item)
	}
	if item.Kind != "song" && item.Kind != "album" && item.Kind != "playlist" {
		return m, nil
	}
	if m.audioInitializing() {
		m.sNotice = audioInitializingText
		return m, nil
	}
	m.searchRequestID++
	m.searchOpen = false
	return m.playDiscoveryTrack(item)
}
func (m model) searchTabLabels() []string {
	counts := []int{len(m.sRes.Recent), len(m.sRes.Songs), len(m.sRes.Albums), len(m.sRes.Playlists), len(m.sRes.Artists)}
	labels := make([]string, len(tabNames))
	for i, name := range tabNames {
		labels[i] = name
		if m.searchGeometry().inner >= 80 {
			labels[i] = fmt.Sprintf("%s %d", name, counts[i])
		}
	}
	return labels
}
func (m model) searchView() string {
	l := m.searchGeometry()
	bright := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	fit := func(text string, width int) string { return pad(ansi.Truncate(artistLabel(text), width, "…"), width) }
	cursor := ""
	if m.sInput && int(m.t*2)%2 == 0 {
		cursor = "▌"
	}
	input := bright.Render(" / ") + dim.Render(ansi.Truncate(artistLabel(m.sQuery), l.inner-5, "…")) + bright.Render(cursor)
	var tabs []string
	for i, label := range m.searchTabLabels() {
		style := faint
		if i == m.sTab {
			style = bright
		}
		tabs = append(tabs, style.Render(label))
	}
	tabLine := " " + strings.Join(tabs, " · ")
	list := m.curList()
	selected := m.selectedSearchItem()
	status := fmt.Sprintf("%s · %d %s", tabNames[m.sTab], len(list), searchResultUnit(len(list)))
	if m.sBusy {
		status = "Searching Apple Music…"
	} else if m.sErr != "" {
		status = m.sErr
	} else if m.sTerm != "" {
		status += " · " + m.sTerm
	} else if strings.TrimSpace(m.sQuery) == "" {
		status = "Find songs, albums, playlists and artists"
	} else if m.sInput {
		status = "Press Enter to search"
	}
	var rows []string
	if m.sBusy || len(list) == 0 {
		text := "Type a name above, then press Enter."
		if m.sTerm != "" && !m.sBusy {
			text = "No matches in this category. Try another tab or query."
		}
		if m.sBusy {
			text = "Loading results…"
		}
		rows = append(rows, faint.Render(fit(text, l.inner)))
	} else {
		start := statsListStart(m.sSel, m.sOffset, l.rows, len(list))
		for i := start; i < min(start+l.rows, len(list)); i++ {
			item := list[i]
			prefix := fmt.Sprintf("  %d. ", i+1)
			style := dim
			if !m.sInput && i == m.sSel {
				prefix = fmt.Sprintf("› %d. ", i+1)
				style = bright
			}
			name := artistLabel(item.Title)
			if item.Artist != "" {
				name += " — " + artistLabel(item.Artist)
			}
			duration := ""
			if item.Duration > 0 {
				duration = " " + fmtTime(item.Duration)
			}
			room := max(0, l.listW-lipgloss.Width(prefix)-lipgloss.Width(duration))
			line := prefix + fit(name, room) + duration
			if !m.sInput && i == m.sSel {
				style = style.Background(selBg)
			}
			rows = append(rows, style.Render(pad(line, l.listW)))
		}
	}
	for len(rows) < l.rows {
		rows = append(rows, strings.Repeat(" ", l.listW))
	}
	body := strings.Join(rows, "\n")
	if len(list) > 0 && !m.sBusy {
		picture := m.artistPicture(selected, l.photoCols, l.photoRows)
		for len(picture) < l.rows {
			picture = append(picture, strings.Repeat(" ", l.photoCols))
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "   ", strings.Join(picture, "\n"))
	}
	detail := ""
	if selected.ID != "" && !m.sBusy {
		detail = selected.Title
		if selected.Artist != "" {
			detail += " — " + selected.Artist
		}
	}
	if m.sNotice != "" {
		detail = m.sNotice
	}
	hints := " Enter search · ↓ browse · Tab category · Esc close"
	if !m.sInput && len(list) > 0 {
		hints = " ↑↓ Select · Enter play · Tab category · / Edit · Esc close"
		if selected.Kind == "artist" || selected.Kind == "album" || selected.Kind == "playlist" {
			hints = " ↑↓ Select · Enter open · Tab category · / Edit · Esc close"
		}
		if selected.Kind == "album" || selected.Kind == "playlist" {
			hints = " Enter open · e Play all · a/A Queue · Tab category · Esc"
		}
	}
	content := pad(input, l.inner) + "\n" + pad(tabLine, l.inner) + "\n\n" + faint.Render(fit(status, l.inner)) + "\n\n" + body + "\n\n" + dim.Render(fit(detail, l.inner)) + "\n" + faint.Render(fit(hints, l.inner))
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentHi).Width(l.inner).Render(content)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, bright.Render("SEARCH APPLE MUSIC")+"\n"+box)
}
func (m model) updateSearchMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	l := m.searchGeometry()
	if msg.X <= l.x || msg.X >= l.x+l.width-1 {
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Y == l.y+2 {
		m.sInput = true
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Y == l.tabY {
		x := l.x + 2
		for i, label := range m.searchTabLabels() {
			if msg.X >= x && msg.X < x+lipgloss.Width(label) {
				m.sTab, m.sSel, m.sOffset = i, 0, 0
				m.sNotice = ""
				return m.requestSearchArtwork()
			}
			x += lipgloss.Width(label) + 3
		}
		return m, nil
	}
	if m.sBusy || msg.Y < l.listY || msg.Y >= l.listY+l.rows {
		return m, nil
	}
	list := m.curList()
	if len(list) == 0 {
		return m, nil
	}
	m.sInput = false
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		m.sSel = max(0, min(m.sSel+step, len(list)-1))
		m.sClickID = ""
		return m.requestSearchArtwork()
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if msg.X >= l.x+1+l.listW {
		return m.activateSearchItem(false)
	}
	start := statsListStart(m.sSel, m.sOffset, l.rows, len(list))
	index := start + msg.Y - l.listY
	if index >= len(list) {
		return m, nil
	}
	m.sSel, m.sOffset = index, start
	item := list[index]
	id, now := item.Kind+":"+item.ID, time.Now()
	if m.sClickID == id && now.Sub(m.sClickAt) < 400*time.Millisecond {
		m.sClickID = ""
		return m.activateSearchItem(false)
	}
	m.sClickID, m.sClickAt = id, now
	return m.requestSearchArtwork()
}

func searchResultUnit(count int) string {
	if count == 1 {
		return "result"
	}
	return "results"
}
