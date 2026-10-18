package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

var tabNames = [5]string{"RECENT", "SONGS", "ALBUMS", "PLAYLISTS", "ARTISTS"}

func (m model) curList() []engine.Track {
	switch m.sTab {
	case 4:
		return m.sRes.Artists
	case 1:
		return m.sRes.Songs
	case 2:
		return m.sRes.Albums
	case 3:
		return m.sRes.Playlists
	}
	return m.sRes.Recent
}

func (m model) searchCmd() tea.Cmd {
	eng, term := m.eng, m.sQuery
	return func() tea.Msg {
		res, err := eng.Search(term)
		return searchMsg{res: res, err: err}
	}
}

func (m model) libraryCmd() tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		res, err := eng.Library()
		return searchMsg{res: res, err: err, keepInput: true}
	}
}

func (m model) discoveryCmd() tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		feed, err := eng.Discovery()
		return discoveryMsg{feed: feed, err: err}
	}
}

func (m model) statsCmd() tea.Cmd {
	return func() tea.Msg { return statsMsg{} }
}

func (m model) openLibrarySelection(selection int) (tea.Model, tea.Cmd) {
	switch selection {
	case 0:
		m.discoveryOpen, m.statsOpen, m.searchOpen = true, false, false
		m.discoveryDetail, m.discoveryBusy = false, true
		m.discoveryErr, m.discoveryTracks, m.discoveryTitle = "", nil, ""
		return m, m.discoveryCmd()
	case 6:
		m.statsOpen, m.discoveryOpen, m.searchOpen = true, false, false
		m.statsBusy, m.statsErr = true, ""
		return m, m.statsCmd()
	default:
		m.libraryCategory = max(0, min(selection-1, 4))
		m.libraryOffset = 0
		m.libraryOpen, m.searchOpen, m.sInput, m.sBusy = true, false, false, true
		m.libraryErr, m.libraryDetailTitle = "", ""
		m.libraryPaged = true
		return m, m.pagedLibraryCmd()
	}
}

type discoveryChoice struct {
	section string
	track   engine.Track
}

func (m model) discoveryItems() []discoveryChoice {
	var items []discoveryChoice
	for _, section := range m.discoveryFeed.Sections {
		for _, track := range section.Items {
			items = append(items, discoveryChoice{section: section.Title, track: track})
		}
	}
	return items
}

func (m model) discoveryItemCount() int { return len(m.discoveryItems()) }

func (m model) discoveryTracksCmd(track engine.Track) tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		if track.Kind == "artist" {
			res, err := eng.ArtistPage(track.ID)
			return discoveryTracksMsg{tracks: res.Songs, err: err}
		}
		tracks, err := eng.DiscoveryTracks(track.Kind, track.ID)
		return discoveryTracksMsg{tracks: tracks, err: err}
	}
}

func (m model) saveDiscoveryCmd(track engine.Track) tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		if err := eng.SaveToLibrary(track.Kind, track.ID); err != nil {
			return noteMsg("could not save to library: " + err.Error())
		}
		return noteMsg("saved to library: " + track.Title)
	}
}

func (m model) queueDiscoveryCmd(track engine.Track, next bool) tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		queue := eng.QueueLater
		if next {
			queue = eng.QueueNext
		}
		if err := queue(track.Kind, track.ID); err != nil {
			return noteMsg("could not queue " + track.Title + ": " + err.Error())
		}
		where := "queue"
		if next {
			where = "next"
		}
		return noteMsg("added to " + where + ": " + track.Title)
	}
}

func (m model) openQueueManager() (tea.Model, tea.Cmd) {
	m.queueOpen, m.queuePrevFocus, m.focus = true, m.focus, focusQueue
	m.selIdx = min(m.selIdx, max(0, len(m.st.Queue)-1))
	return m, nil
}

func (m model) closeQueueManager() (tea.Model, tea.Cmd) {
	m.queueOpen, m.focus = false, m.queuePrevFocus
	return m, nil
}

func (m model) updateQueueManager(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	eng := m.eng
	if direction := pageDirection(msg); direction != 0 {
		m.selIdx = max(0, min(m.selIdx+direction*max(1, m.searchRows()), len(m.st.Queue)-1))
		return m, nil
	}
	switch msg.String() {
	case "esc", "q":
		return m.closeQueueManager()
	case "up", "k":
		m.selIdx = max(0, m.selIdx-1)
	case "down", "j":
		m.selIdx = min(m.selIdx+1, max(0, len(m.st.Queue)-1))
	case "enter":
		if m.selIdx < len(m.st.Queue) {
			i := m.selIdx
			m.note, m.noteAt = "playing: "+m.st.Queue[i].Title, m.t
			return m, doCmd(func() error { return eng.JumpTo(i) })
		}
	case "delete", "backspace":
		if m.selIdx < len(m.st.Queue) {
			track := m.st.Queue[m.selIdx]
			i := m.selIdx
			m.note, m.noteAt = "removed from queue: "+track.Title, m.t
			return m, doCmd(func() error { return eng.EditQueue("remove", i, 0) })
		}
	case "ctrl+up", "ctrl+down":
		if m.selIdx < len(m.st.Queue) {
			from := m.selIdx
			to := from + 1
			if msg.String() == "ctrl+up" {
				to = from - 1
			}
			if to >= 0 && to < len(m.st.Queue) {
				m.selIdx = to
				m.note, m.noteAt = "moved "+m.st.Queue[from].Title, m.t
				return m, doCmd(func() error { return eng.EditQueue("move", from, to) })
			}
		}
	case "ctrl+delete":
		if len(m.st.Queue) > 0 {
			m.selIdx = 0
			m.note, m.noteAt = "cleared queue", m.t
			return m, doCmd(func() error { return eng.EditQueue("clear", 0, 0) })
		}
	}
	return m, nil
}

func (m model) queueManagerView() string {
	iw := min(100, m.w-10) - 2
	rows := max(5, min(20, m.h-13))
	title := fmt.Sprintf("QUEUE · %d TRACKS", len(m.st.Queue))
	if len(m.st.Queue) > 0 && m.st.QueuePos >= 0 && m.st.QueuePos < len(m.st.Queue) {
		title += fmt.Sprintf(" · NOW %d/%d", m.st.QueuePos+1, len(m.st.Queue))
	}
	body := m.queuePanel(iw, rows+1)
	body += "\n" + lipgloss.NewStyle().Foreground(fgFaint).Render(" ↑/↓ select · enter play · del remove · ctrl+↑/↓ reorder · ctrl+del clear · esc/q back")
	box := panel(title, body, iw, rows+2, true, accentHi)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("POMME · QUEUE MANAGER")+"\n"+box)
}

func (m model) playDiscoveryTrack(track engine.Track) (tea.Model, tea.Cmd) {
	if m.audioInitializing() {
		m.note, m.noteAt = audioInitializingText, m.t
		return m, nil
	}
	m.searchOpen, m.discoveryOpen = false, false
	m.loading, m.loadSnap, m.loadStart = track.Title, snap(m.st), m.t
	m.loadHideQueue = true
	m.beginAudioInitialization()
	kind, id := track.Kind, track.ID
	return m, doCmd(func() error { return m.eng.Play(kind, id) })
}

func (m model) updateDiscovery(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.saveConfirmOpen {
		switch msg.String() {
		case "y", "Y", "enter":
			track := m.saveConfirmTrack
			m.saveConfirmOpen, m.saveConfirmTrack = false, engine.Track{}
			m.note, m.noteAt = "saving to library: "+track.Title, m.t
			return m, m.saveDiscoveryCmd(track)
		case "n", "N", "esc", "ctrl+c":
			m.saveConfirmOpen, m.saveConfirmTrack = false, engine.Track{}
			m.note, m.noteAt = "save cancelled", m.t
		}
		return m, nil
	}
	if direction := pageDirection(msg); direction != 0 {
		if m.discoveryDetail {
			m.discoveryTrackSel = max(0, min(m.discoveryTrackSel+direction*m.searchRows(), len(m.discoveryTracks)-1))
		} else {
			m.discoverySel = max(0, min(m.discoverySel+direction*m.searchRows(), m.discoveryItemCount()-1))
		}
		return m, nil
	}
	switch msg.String() {
	case "esc", "ctrl+c":
		if m.discoveryDetail {
			m.discoveryDetail, m.discoveryBusy = false, false
			m.discoveryErr, m.discoveryTracks, m.discoveryTitle = "", nil, ""
			return m, nil
		}
		m.discoveryOpen = false
		return m.requestRecentArtwork()
	case "q":
		return m.openQueueManager()
	case "up", "k":
		if m.discoveryDetail {
			m.discoveryTrackSel = max(0, m.discoveryTrackSel-1)
		} else {
			m.discoverySel = max(0, m.discoverySel-1)
		}
	case "down", "j":
		if m.discoveryDetail {
			m.discoveryTrackSel = min(m.discoveryTrackSel+1, max(0, len(m.discoveryTracks)-1))
		} else {
			m.discoverySel = min(m.discoverySel+1, max(0, m.discoveryItemCount()-1))
		}
	case "enter":
		if m.discoveryDetail {
			if len(m.discoveryTracks) > 0 {
				return m.playDiscoveryTrack(m.discoveryTracks[m.discoveryTrackSel])
			}
			return m, nil
		}
		items := m.discoveryItems()
		if m.discoverySel >= len(items) {
			return m, nil
		}
		item := items[m.discoverySel].track
		if item.Kind == "song" {
			return m.playDiscoveryTrack(item)
		}
		if item.Kind != "album" && item.Kind != "playlist" && item.Kind != "artist" && item.Kind != "genre" {
			m.note, m.noteAt = "this recommendation cannot be opened", m.t
			return m, nil
		}
		m.discoveryBusy, m.discoveryDetail, m.discoveryTitle = true, true, item.Title
		return m, m.discoveryTracksCmd(item)
	case "e", "E":
		if m.discoveryDetail {
			if len(m.discoveryTracks) > 0 {
				return m.playDiscoveryTrack(m.discoveryTracks[m.discoveryTrackSel])
			}
			return m, nil
		}
		items := m.discoveryItems()
		if m.discoverySel < len(items) {
			item := items[m.discoverySel].track
			if item.Kind == "song" || item.Kind == "album" || item.Kind == "playlist" {
				return m.playDiscoveryTrack(item)
			}
			if item.Kind == "genre" {
				m.discoveryBusy, m.discoveryDetail, m.discoveryTitle = true, true, item.Title
				return m, m.discoveryTracksCmd(item)
			}
			m.note, m.noteAt = "this recommendation cannot be played directly", m.t
		}
	case "a", "A":
		var selected engine.Track
		if m.discoveryDetail {
			if m.discoveryTrackSel < len(m.discoveryTracks) {
				selected = m.discoveryTracks[m.discoveryTrackSel]
			}
		} else {
			items := m.discoveryItems()
			if m.discoverySel < len(items) {
				selected = items[m.discoverySel].track
			}
		}
		if selected.ID == "" {
			return m, nil
		}
		if selected.Kind != "song" && selected.Kind != "album" && selected.Kind != "playlist" {
			m.note, m.noteAt = "only songs, albums, and playlists can be queued", m.t
			return m, nil
		}
		where := "queue"
		if msg.String() == "A" {
			where = "next"
		}
		m.note, m.noteAt = "adding to "+where+": "+selected.Title, m.t
		return m, m.queueDiscoveryCmd(selected, msg.String() == "A")
	case "w", "W":
		if m.discoveryDetail {
			if m.discoveryTrackSel < len(m.discoveryTracks) {
				m.saveConfirmOpen, m.saveConfirmTrack = true, m.discoveryTracks[m.discoveryTrackSel]
			}
			return m, nil
		}
		items := m.discoveryItems()
		if m.discoverySel < len(items) {
			track := items[m.discoverySel].track
			if track.Kind != "song" && track.Kind != "album" && track.Kind != "playlist" {
				m.note, m.noteAt = "this recommendation cannot be saved to your library", m.t
				return m, nil
			}
			m.saveConfirmOpen, m.saveConfirmTrack = true, track
			m.note, m.noteAt = "confirm save to library", m.t
			return m, nil
		}
	case "P":
		if m.discoveryDetail {
			items := m.discoveryItems()
			if m.discoverySel < len(items) {
				track := items[m.discoverySel].track
				if track.Kind == "album" || track.Kind == "playlist" {
					m.note, m.noteAt = "adding to queue: "+track.Title, m.t
					return m, m.queueDiscoveryCmd(track, false)
				}
			}
		}
	}
	return m, nil
}

func (m model) updateStats(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.statsOpen = false
		return m.requestRecentArtwork()
	case "r", "R":
		m.listenHistory = loadListeningHistory()
		m.statsErr = ""
	case "left", "h":
		m.statsRange = (m.statsRange + statsRange(len(statsRangeNames)) - 1) % statsRange(len(statsRangeNames))
	case "right", "l":
		m.statsRange = (m.statsRange + 1) % statsRange(len(statsRangeNames))
	}
	return m, nil
}

func (m model) discoveryView() string {
	if m.saveConfirmOpen {
		w := min(72, m.w-8) - 2
		pink := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
		faint := lipgloss.NewStyle().Foreground(fgFaint)
		body := "Add “" + m.saveConfirmTrack.Title + "” to your Apple Music library?\n\n" +
			pink.Render("y / Enter") + faint.Render(" confirm   ") + pink.Render("n / Esc") + faint.Render(" cancel")
		box := panel("SAVE TO LIBRARY?", body, w, 4, true, accentHi)
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
			pink.Render("POMME · DISCOVERY")+"\n"+box)
	}
	bw := min(100, m.w-8)
	iw := bw - 2
	listRows := max(5, min(m.h-15, m.h*55/100))
	var source []string
	selectedLine := -1
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	pink := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	if m.discoveryDetail {
		if m.discoveryBusy {
			source = []string{faint.Render(" loading tracks…")}
		} else if m.discoveryErr != "" {
			source = []string{faint.Render(" " + m.discoveryErr)}
		} else if len(m.discoveryTracks) == 0 {
			source = []string{faint.Render(" no playable tracks found")}
		} else {
			for i, track := range m.discoveryTracks {
				mark, style := "  ", faint
				if i == m.discoveryTrackSel {
					mark, style, selectedLine = "› ", pink, len(source)
				}
				line := mark + track.Title
				if track.Artist != "" {
					line += " · " + track.Artist
				}
				source = append(source, style.Render(line))
			}
		}
	} else if m.discoveryBusy {
		source = []string{faint.Render(" loading recommendations from Apple Music…")}
	} else if m.discoveryErr != "" {
		source = []string{faint.Render(" " + m.discoveryErr)}
	} else {
		choiceIndex := 0
		for _, section := range m.discoveryFeed.Sections {
			source = append(source, lipgloss.NewStyle().Foreground(fgDim).Bold(true).Render(" "+section.Title))
			for _, track := range section.Items {
				mark, style := "  ", faint
				if choiceIndex == m.discoverySel {
					mark, style, selectedLine = "› ", pink, len(source)
				}
				line := mark + track.Title
				if track.Artist != "" {
					line += " · " + track.Artist
				}
				source = append(source, style.Render(line))
				choiceIndex++
			}
		}
		if len(source) == 0 {
			source = []string{faint.Render(" no recommendations are available")}
		}
	}
	start := max(0, selectedLine-listRows+1)
	body := make([]string, 0, listRows)
	for i := start; i < min(start+listRows, len(source)); i++ {
		line := source[i]
		if i == selectedLine {
			line = lipgloss.NewStyle().Background(selBg).Render(line)
		}
		body = append(body, pad(line, iw))
	}
	for len(body) < listRows {
		body = append(body, pad("", iw))
	}
	upperTitle := "DISCOVER"
	if m.discoveryDetail {
		upperTitle = "DISCOVER · " + m.discoveryTitle
	}
	upper := panel(upperTitle, strings.Join(body, "\n"), iw, listRows+1, true, accentHi)

	var info []string
	hints := []string{"↑/↓ Move · Enter Open · e Play", "a Queue · A Next · w Save (confirm)", "q Queue manager · Esc Back"}
	if m.discoveryDetail {
		hints = []string{"↑/↓ Move · Enter/e Play · a Queue song", "A Queue next · P Queue whole list · w Save", "q Queue manager · Esc Back"}
		if len(m.discoveryTracks) > 0 && m.discoveryTrackSel < len(m.discoveryTracks) {
			track := m.discoveryTracks[m.discoveryTrackSel]
			info = append(info, pink.Render(track.Title))
			if track.Artist != "" {
				info = append(info, faint.Render(track.Artist))
			}
		}
	} else {
		items := m.discoveryItems()
		if m.discoverySel < len(items) {
			item := items[m.discoverySel]
			info = append(info, pink.Render(item.track.Title))
			if item.track.Artist != "" {
				info = append(info, faint.Render(item.track.Artist))
			}
			info = append(info, faint.Render(item.section+" · "+item.track.Kind))
		}
	}
	for _, hint := range hints {
		info = append(info, faint.Render(hint))
	}
	detail := panel("SELECTED", strings.Join(info, "\n"), iw, max(4, len(info)+1), false, accent)
	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("POMME · DISCOVERY"), upper, detail)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, content)
}

func (m model) statsView() string {
	bw := min(120, m.w-8)
	iw := max(20, bw-2)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	dim := lipgloss.NewStyle().Foreground(fgDim)
	pink := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	green := lipgloss.NewStyle().Foreground(accentHi)
	now := time.Now()
	history := m.listenHistory
	if history == nil {
		history = &listeningHistory{}
	}
	summary := history.summary(m.statsRange, now)
	var rangeLabels []string
	for i, name := range statsRangeNames {
		if iw < 75 {
			name = [...]string{"7 Days", "30 Days", "Month", "Year", "All"}[i]
		}
		label := name
		if statsRange(i) == m.statsRange {
			label = pink.Render(label)
		} else {
			label = dim.Render(label)
		}
		rangeLabels = append(rangeLabels, label)
	}
	compact := m.h < 36
	selector := panel("STATS · ←/→ RANGE", strings.Join(rangeLabels, "  |  "), iw, 3, true, accentHi)
	minsToday := summary.TodaySeconds / 60
	minsTotal := summary.Seconds / 60
	streak := fmt.Sprintf("%d-day streak (best %d)", summary.Streak, summary.Best)
	if summary.Streak == 0 {
		streak = fmt.Sprintf("0-day streak (best %d)", summary.Best)
	}
	dateSummary := fmt.Sprintf("%d plays · %dh %02dm listened", summary.Plays, minsTotal/60, minsTotal%60)
	if compact {
		dateSummary = fmt.Sprintf("%d plays · %dh %02dm", summary.Plays, minsTotal/60, minsTotal%60)
	} else {
		dateSummary += fmt.Sprintf(" · %dm today", minsToday)
	}
	status := panel(statsRangeNames[m.statsRange], pink.Render(streak)+dim.Render("  ·  "+dateSummary), iw, 3, false, accent)

	listRows, panelHeight := 5, 8
	if compact {
		listRows, panelHeight = 2, 3
	}
	trackWidth := max(16, (iw-4)*42/100)
	artistWidth := max(16, (iw-4)*29/100)
	albumWidth := max(16, iw-4-trackWidth-artistWidth)
	tracksPanel := panel("TOP TRACKS", statsTopLines(summary.Tracks, "track", trackWidth-2, listRows), trackWidth, panelHeight, true, accentHi)
	artistsPanel := panel("TOP ARTISTS", statsTopLines(summary.Artists, "artist", artistWidth-2, listRows), artistWidth, panelHeight, false, accent)
	albumsPanel := panel("TOP ALBUMS", statsTopLines(summary.Albums, "album", albumWidth-2, listRows), albumWidth, panelHeight, false, accent)
	top := lipgloss.JoinHorizontal(lipgloss.Top, tracksPanel, " ", artistsPanel, " ", albumsPanel)

	chartWidth := iw
	chartDays := 10
	if compact {
		chartDays = 3
	}
	chartRows := make([]string, 0, chartDays)
	maxSeconds := 0
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := chartDays - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		if summary.Days[day] > maxSeconds {
			maxSeconds = summary.Days[day]
		}
	}
	barWidth := max(4, chartWidth-22)
	for i := chartDays - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		day := date.Format("2006-01-02")
		seconds := summary.Days[day]
		filled := 0
		if maxSeconds > 0 {
			filled = seconds * barWidth / maxSeconds
		}
		bar := green.Render(strings.Repeat("█", filled)) + dim.Render(strings.Repeat("░", max(0, barWidth-filled)))
		chartRows = append(chartRows, fmt.Sprintf("%s  %s  %s", date.Format("2006-01-02"), bar, faint.Render(formatListenDuration(seconds))))
	}
	chartTitle := fmt.Sprintf("LAST %d DAYS · POMME HISTORY", chartDays)
	chartHeight := len(chartRows) + 2
	if compact {
		chartHeight = len(chartRows) + 1
	}
	chart := panel(chartTitle, strings.Join(chartRows, "\n"), chartWidth, chartHeight, false, accent)
	content := lipgloss.JoinVertical(lipgloss.Left, selector, status, top, chart)
	if !compact {
		content = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("POMME · STATS"),
			content,
			faint.Render(" ←/→ change range · r refresh · Esc close · local listening history"))
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, content)
}

func statsTopLines(records []listeningRecord, kind string, width, rows int) string {
	if width < 1 {
		return ""
	}
	var lines []string
	if len(records) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(fgFaint).Render(" no plays yet"))
	} else {
		for i, record := range records {
			name := record.Title
			if kind == "track" && record.Artist != "" {
				name += " — " + record.Artist
			}
			line := fmt.Sprintf("%d. %s  %d plays", i+1, name, record.Plays)
			if i == 0 {
				line = lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("› " + line)
			}
			lines = append(lines, pad(line, width))
		}
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func formatListenDuration(seconds int) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%dh %02dm", seconds/3600, seconds%3600/60)
	}
	if seconds > 0 && seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm", seconds/60)
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	eng := m.eng
	list := m.curList()
	if direction := pageDirection(msg); direction != 0 {
		if m.sBusy {
			return m, nil
		}
		m.sInput = false
		atBoundary := (direction < 0 && m.sSel == 0) || (direction > 0 && m.sSel >= len(list)-1)
		if m.libraryPaged && atBoundary && ((direction < 0 && m.libraryOffset > 0) || (direction > 0 && len(list) == 50 && m.libraryOffset < 10000)) {
			m.libraryOffset = max(0, m.libraryOffset+direction*50)
			m.sBusy = true
			return m, m.pagedLibraryCmd()
		}
		m.sSel = max(0, min(m.sSel+direction*m.searchRows(), len(list)-1))
		return m, nil
	}
	switch msg.String() {
	case "L":
		if m.sInput {
			m.sQuery += "L"
			return m, nil
		}
		return m.openLibrarySelection(m.libraryCategory + 1)
	case "ctrl+p", "ctrl+n":
		if len(m.searchHistory) > 0 {
			if msg.String() == "ctrl+p" {
				m.historyIndex = min(m.historyIndex+1, len(m.searchHistory)-1)
			} else {
				m.historyIndex = max(m.historyIndex-1, 0)
			}
			m.sQuery = m.searchHistory[m.historyIndex]
			m.sInput = true
		}
	case "esc", "ctrl+c":
		m.searchOpen = false
		return m.requestRecentArtwork()
	case "enter":
		if m.sInput {
			if strings.TrimSpace(m.sQuery) != "" {
				m.searchHistory = rememberSearch(m.searchHistory, m.sQuery)
				saveSearchHistory(m.searchHistory)
				m.historyIndex = -1
				m.sBusy = true
				return m, m.searchCmd()
			}
			m.sBusy = true // empty query = reload library
			return m, m.libraryCmd()
		} else if m.sSel < len(list) {
			if m.audioInitializing() {
				m.note, m.noteAt = audioInitializingText, m.t
				return m, nil
			}
			kind, id, title := list[m.sSel].Kind, list[m.sSel].ID, list[m.sSel].Title
			if kind == "artist" {
				m.sBusy = true
				m.sTab = 1
				return m, func() tea.Msg { res, err := eng.ArtistPage(id); return searchMsg{res: res, err: err} }
			}
			m.searchOpen = false
			m.loading, m.loadSnap, m.loadStart = title, snap(m.st), m.t
			m.loadHideQueue = true // the whole queue is being replaced
			m.beginAudioInitialization()
			play := doCmd(func() error { return eng.Play(kind, id) })
			m, art := m.requestRecentArtwork()
			return m, tea.Batch(play, art)
		}
	case "tab":
		m.sTab, m.sSel = (m.sTab+1)%len(tabNames), 0
	case "/":
		m.sInput = true
	case "backspace":
		m.sInput = true
		if len(m.sQuery) > 0 {
			r := []rune(m.sQuery)
			m.sQuery = string(r[:len(r)-1])
		}
	case "up":
		if m.sInput {
			m.sInput = false // arrow dives from input into the list
		} else {
			m.sSel = max(m.sSel-1, 0)
		}
	case "down":
		if m.sInput {
			m.sInput, m.sSel = false, 0
		} else {
			m.sSel = min(m.sSel+1, max(0, len(list)-1))
		}
	case "k":
		if m.sInput {
			m.sQuery += "k"
		} else {
			m.sSel = max(m.sSel-1, 0)
		}
	case "j":
		if m.sInput {
			m.sQuery += "j"
		} else {
			m.sSel = min(m.sSel+1, max(0, len(list)-1))
		}
	case "a", "A":
		if !m.sInput {
			if m.sSel < len(list) {
				kind, id := list[m.sSel].Kind, list[m.sSel].ID
				f := eng.QueueLater
				m.note, m.noteAt = "added to queue: "+list[m.sSel].Title, m.t
				if msg.String() == "A" {
					f = eng.QueueNext
					m.note = "playing next: " + list[m.sSel].Title
				}
				return m, doCmd(func() error { return f(kind, id) })
			}
		} else {
			m.sQuery += msg.String()
		}
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.sInput = true // typing anywhere returns focus to the input
			m.sQuery += string(msg.Runes)
			if msg.Type == tea.KeySpace {
				m.sQuery += " "
			}
		}
	}
	return m, nil
}

// Library categories have their own browsing page. The slash key remains a
// deliberate handoff to catalog search so library navigation never hijacks it.
func (m model) updateLibraryView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	eng := m.eng
	list := m.curList()
	if direction := pageDirection(msg); direction != 0 {
		if m.sBusy {
			return m, nil
		}
		atBoundary := (direction < 0 && m.sSel == 0) || (direction > 0 && m.sSel >= len(list)-1)
		if m.libraryPaged && atBoundary && ((direction < 0 && m.libraryOffset > 0) || (direction > 0 && len(list) == 50 && m.libraryOffset < 10000)) {
			m.libraryOffset = max(0, m.libraryOffset+direction*50)
			m.sBusy = true
			return m, m.pagedLibraryCmd()
		}
		m.sSel = max(0, min(m.sSel+direction*m.searchRows(), len(list)-1))
		return m, nil
	}
	switch msg.String() {
	case "esc", "ctrl+c":
		if m.libraryDetailTitle != "" {
			m.libraryDetailTitle, m.libraryPaged = "", true
			m.libraryOffset, m.sSel, m.sBusy = 0, 0, true
			return m, m.pagedLibraryCmd()
		}
		m.libraryOpen = false
		return m.requestRecentArtwork()
	case "tab":
		m.libraryCategory = (m.libraryCategory + 1) % 5
		m.libraryOffset, m.sSel = 0, 0
		m.libraryDetailTitle, m.libraryErr = "", ""
		m.libraryPaged, m.sBusy = true, true
		m.sTab = []int{1, 1, 2, 3, 4}[m.libraryCategory]
		return m, m.pagedLibraryCmd()
	case "/":
		m.libraryOpen, m.searchOpen, m.sInput = false, true, true
		return m, nil
	case "enter":
		if m.sBusy || m.sSel >= len(list) || len(list) == 0 {
			return m, nil
		}
		selected := list[m.sSel]
		if selected.Kind == "artist" {
			m.libraryDetailTitle = selected.Title
			m.libraryPaged, m.sBusy = false, true
			m.sTab, m.sSel = 1, 0
			return m, func() tea.Msg {
				res, err := eng.ArtistPage(selected.ID)
				return searchMsg{res: res, err: err}
			}
		}
		if m.audioInitializing() {
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
		m.libraryOpen = false
		m.loading, m.loadSnap, m.loadStart = selected.Title, snap(m.st), m.t
		m.loadHideQueue = true
		m.beginAudioInitialization()
		play := doCmd(func() error { return eng.Play(selected.Kind, selected.ID) })
		m, art := m.requestRecentArtwork()
		return m, tea.Batch(play, art)
	case "a", "A":
		if !m.sBusy && m.sSel < len(list) {
			selected := list[m.sSel]
			queue := eng.QueueLater
			m.note, m.noteAt = "added to queue: "+selected.Title, m.t
			if msg.String() == "A" {
				queue = eng.QueueNext
				m.note = "playing next: " + selected.Title
			}
			return m, doCmd(func() error { return queue(selected.Kind, selected.ID) })
		}
	case "up", "k":
		m.sSel = max(0, m.sSel-1)
	case "down", "j":
		m.sSel = min(m.sSel+1, max(0, len(list)-1))
	}
	return m, nil
}

func (m model) libraryView() string {
	names := [5]string{"Songs", "Albums", "Playlists", "Artists", "Favorites"}
	title := strings.ToUpper(names[m.libraryCategory])
	if m.libraryDetailTitle != "" {
		title = "ARTIST · " + m.libraryDetailTitle
	}
	bw := min(100, m.w-8)
	iw := bw - 2
	rows := max(4, min(18, m.h-13))
	pink := lipgloss.NewStyle().Foreground(accentHi)
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	list := m.curList()
	var body []string
	switch {
	case m.sBusy:
		body = append(body, faint.Render(" Loading your "+strings.ToLower(title)+"…"))
	case m.libraryErr != "":
		body = append(body, lipgloss.NewStyle().Foreground(accentHi).Render(" "+m.libraryErr))
	case len(list) == 0:
		body = append(body, faint.Render(" Nothing here yet."))
	default:
		start := max(0, m.sSel-rows+1)
		for i := start; i < min(start+rows, len(list)); i++ {
			track := list[i]
			line := "  " + track.Title + dim.Render("  ·  "+track.Artist)
			if track.Duration > 0 {
				line += faint.Render("  " + fmtTime(track.Duration))
			}
			if i == m.sSel {
				line = lipgloss.NewStyle().Background(selBg).Render(pad(pink.Render("›")+line, iw))
			}
			body = append(body, pad(line, iw))
		}
	}
	for len(body) < rows {
		body = append(body, "")
	}
	page := ""
	if m.libraryPaged {
		page = fmt.Sprintf("  ·  page %d", m.libraryOffset/50+1)
	}
	body = append(body, faint.Render(" ↑/↓ Move · Enter Play/open"))
	body = append(body, faint.Render(" a Queue · A Play next · PgUp/PgDn Page"))
	body = append(body, faint.Render(" Tab Category · / Search · Esc Back"))
	content := panel(title+page, strings.Join(body, "\n"), iw, len(body)+1, true, accentHi)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		pink.Bold(true).Render("POMME · LIBRARY")+"\n"+content)
}

func (m model) searchView() string {
	bw := min(84, m.w-8)
	iw := bw - 2 // inner width

	pink := lipgloss.NewStyle().Foreground(accentHi)
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)

	cursor := " "
	if m.sInput {
		cursor = pink.Render("▊")
	}
	input := pink.Render(" / ") + lipgloss.NewStyle().Foreground(fgBright).Render(m.sQuery) + cursor

	var tabs []string
	for i, name := range tabNames {
		st := faint
		if i == m.sTab {
			st = pink.Bold(true)
		}
		tabs = append(tabs, st.Render(name))
	}
	tabLine := " " + strings.Join(tabs, faint.Render("  ·  "))

	rows := m.searchRows()
	list := m.curList()
	var body []string
	switch {
	case m.sBusy:
		body = append(body, dim.Render(" searching…"))
	case len(list) == 0 && strings.TrimSpace(m.sQuery) != "":
		body = append(body, faint.Render(" no results — enter to search"))
	case len(list) == 0:
		body = append(body, faint.Render(" your library — albums & playlists tabs · type to search the catalog"))
	default:
		start := max(0, m.sSel-rows+1)
		for i := start; i < min(start+rows, len(list)); i++ {
			tr := list[i]
			line := dim.Render(" "+tr.Title) + faint.Render(" — "+tr.Artist)
			if tr.Duration > 0 {
				line += faint.Render("  " + fmtTime(tr.Duration))
			}
			if !m.sInput && i == m.sSel {
				line = lipgloss.NewStyle().Background(selBg).Render(pad(pink.Render(" ")+line, iw))
			}
			body = append(body, pad(line, iw))
		}
	}
	hint := faint.Render(" ↵ play · PgUp/Dn or Ctrl+B/F page · tab category · esc close")
	if m.note != "" {
		hint = dim.Render(" " + m.note)
	}

	content := pad(input, iw) + "\n" + pad(tabLine, iw) + "\n\n" +
		strings.Join(body, "\n") + "\n\n" + pad(hint, iw)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(accent).
		Width(iw).Render(content)
	title := pink.Bold(true).Render("SEARCH APPLE MUSIC")
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, title+"\n"+box)
}

func (m model) pagedLibraryCmd() tea.Cmd {
	categories := []string{"songs", "albums", "playlists", "artists", "favorites"}
	category := categories[m.libraryCategory]
	offset := m.libraryOffset
	eng := m.eng
	return func() tea.Msg {
		res, err := eng.LibraryPage(category, offset)
		return searchMsg{res: res, err: err, category: category, offset: offset}
	}
}
