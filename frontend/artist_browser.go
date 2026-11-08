package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type artistOrigin int

const (
	artistFromLibrary artistOrigin = iota
	artistFromStats
	artistFromPlaylists
	artistFromAlbums
	artistFromStatsAlbum
	artistFromSearch
)

type artistMode int

const (
	artistList artistMode = iota
	artistDetail
	artistAlbum
	artistResolving
)

type artistLoad int

const (
	artistLoadList artistLoad = iota
	artistLoadProfile
	artistLoadAlbum
)

type artistGridState struct {
	items                 []engine.Track
	selected, start, page int
}
type artistSongState struct {
	items           []engine.Track
	selected, start int
}
type artistBrowserState struct {
	open, busy                         bool
	origin                             artistOrigin
	mode                               artistMode
	err, notice, lookupKey, lookupName string
	root, albums                       artistGridState
	profile, album                     engine.Track
	songs, albumSongs                  artistSongState
	tab                                int
}
type artistBrowserMsg struct {
	request uint64
	kind    artistLoad
	results engine.SearchResults
	tracks  []engine.Track
	err     error
}

type artistQueueMsg struct {
	request uint64
	notice  string
	err     error
}

type artistGeometry struct{ x, y, width, contentH, gridY, cardW, cardH, cols, gridRows, start, thumbRows, heroRows, listY, listRows, tabsY int }

func (m model) artistGeometry() artistGeometry {
	b := m.artistBrowser
	l := artistGeometry{width: min(112, m.w-8)}
	l.cols = min(3, max(1, (l.width+2)/28))
	l.cardW = (l.width - 2*(l.cols-1)) / l.cols
	l.width = l.cols*l.cardW + 2*(l.cols-1)
	if b.mode == artistList || b.mode == artistResolving {
		l.thumbRows = min(12, max(6, (l.cardW-6)/2))
		l.cardH = l.thumbRows + 4
		l.gridRows = max(1, min(2, (m.h-3)/(l.cardH+1)))
		l.contentH = l.gridRows*l.cardH + l.gridRows - 1 + 4
		l.start = artistGridStart(b.root, l.cols, l.gridRows)
		l.y = max(0, (m.h-l.contentH)/2)
		l.gridY = l.y + 2
	} else {
		l.heroRows = min(12, max(4, (m.h-12)/2))
		// Heading, portrait, blank line, tabs, blank line and two hint rows.
		fixed := l.heroRows + 6
		available := max(3, m.h-fixed)
		l.thumbRows = min(6, max(3, (available-4)/2))
		l.cardH = l.thumbRows + 4
		l.gridRows = max(1, min(2, (available+1)/(l.cardH+1)))
		if b.mode == artistDetail && b.tab == 1 {
			bodyH := l.gridRows*l.cardH + l.gridRows - 1
			l.contentH = fixed + bodyH
			l.start = artistGridStart(b.albums, l.cols, l.gridRows)
		} else {
			l.listRows = min(12, available)
			l.contentH = fixed + l.listRows
		}
		l.y = max(0, (m.h-l.contentH)/2)
		l.tabsY = l.y + 1 + l.heroRows + 1
		l.gridY = l.tabsY + 2
		l.listY = l.gridY
	}
	l.x = max(0, (m.w-l.width)/2)
	return l
}

func artistGridStart(grid artistGridState, cols, rows int) int {
	selected := max(0, min(grid.selected, len(grid.items)-1))
	totalRows := (len(grid.items) + cols - 1) / cols
	start := max(0, min(grid.start/cols, max(0, totalRows-rows)))
	if selected/cols < start {
		start = selected / cols
	}
	if selected/cols >= start+rows {
		start = selected/cols - rows + 1
	}
	return start * cols
}

func (m model) openArtistLibrary(page int) (tea.Model, tea.Cmd) {
	m.artistRequestID++
	m.artistBrowser.open, m.artistBrowser.busy = true, true
	m.artistBrowser.mode, m.artistBrowser.err = artistList, ""
	m.artistBrowser.root = artistGridState{page: page}
	category := "artists"
	if m.artistBrowser.origin == artistFromPlaylists {
		category = "playlists"
	} else if m.artistBrowser.origin == artistFromAlbums {
		category = "albums"
	}
	request, eng := m.artistRequestID, m.eng
	if eng == nil {
		m.artistBrowser.busy = false
		m.artistBrowser.err = "Connect to Apple Music to browse " + category + "."
		return m, nil
	}
	return m, func() tea.Msg {
		res, err := eng.LibraryPage(category, page)
		return artistBrowserMsg{request: request, kind: artistLoadList, results: res, err: err}
	}
}

func (m model) openArtistProfile(profile engine.Track) (tea.Model, tea.Cmd) {
	m.artistRequestID++
	m.artistBrowser.open, m.artistBrowser.busy = true, true
	m.artistBrowser.mode, m.artistBrowser.profile = artistDetail, profile
	m.artistBrowser.err, m.artistBrowser.notice = "", ""
	m.artistBrowser.songs, m.artistBrowser.albums = artistSongState{}, artistGridState{}
	m.artistBrowser.tab = 0
	request, eng := m.artistRequestID, m.eng
	if eng == nil {
		m.artistBrowser.busy = false
		m.artistBrowser.err = "Connect to Apple Music to open this artist."
		return m, nil
	}
	return m, func() tea.Msg {
		res, err := eng.ArtistPage(profile.ID)
		return artistBrowserMsg{request: request, kind: artistLoadProfile, results: res, err: err}
	}
}

func (m model) openArtistAlbum(album engine.Track) (tea.Model, tea.Cmd) {
	m.artistRequestID++
	m.artistBrowser.mode, m.artistBrowser.album = artistAlbum, album
	m.artistBrowser.busy, m.artistBrowser.err, m.artistBrowser.notice = true, "", ""
	m.artistBrowser.albumSongs = artistSongState{}
	request, eng := m.artistRequestID, m.eng
	return m, func() tea.Msg {
		tracks, err := eng.DiscoveryTracks(album.Kind, album.ID)
		return artistBrowserMsg{request: request, kind: artistLoadAlbum, tracks: tracks, err: err}
	}
}

func (m model) applyArtistBrowser(msg artistBrowserMsg) (tea.Model, tea.Cmd) {
	if !m.artistBrowser.open || msg.request != m.artistRequestID {
		return m, nil
	}
	m.sBusy = false
	m.artistBrowser.busy = false
	if msg.err != nil {
		m.artistBrowser.err = msg.err.Error()
		return m, nil
	}
	switch msg.kind {
	case artistLoadList:
		m.artistBrowser.root.items = msg.results.Artists
		if m.artistBrowser.origin == artistFromPlaylists {
			m.artistBrowser.root.items = msg.results.Playlists
		} else if m.artistBrowser.origin == artistFromAlbums {
			m.artistBrowser.root.items = msg.results.Albums
		}
	case artistLoadProfile:
		if len(msg.results.Artists) > 0 {
			m.artistBrowser.profile = msg.results.Artists[0]
		}
		m.artistBrowser.songs.items = msg.results.Songs
		m.artistBrowser.albums.items = msg.results.Albums
	case artistLoadAlbum:
		m.artistBrowser.albumSongs.items = msg.tracks
	}
	return m.requestVisibleArtistImages()
}

func (m model) requestVisibleArtistImages() (model, tea.Cmd) {
	if !m.artistBrowser.open || m.artistBrowser.busy {
		return m, nil
	}
	l := m.artistGeometry()
	b := m.artistBrowser
	var items []engine.Track
	switch b.mode {
	case artistList:
		items = b.root.items[l.start:min(l.start+l.cols*l.gridRows, len(b.root.items))]
	case artistDetail:
		items = append(items, b.profile)
		if b.tab == 1 {
			items = append(items, b.albums.items[l.start:min(l.start+l.cols*l.gridRows, len(b.albums.items))]...)
		}
	case artistAlbum:
		items = append(items, b.album)
	}
	return m.requestArtistImages(items)
}

func (m model) artistBack() (tea.Model, tea.Cmd) {
	m.artistRequestID++
	m.sBusy = false
	m.artistBrowser.busy, m.artistBrowser.err, m.artistBrowser.notice = false, "", ""
	switch m.artistBrowser.mode {
	case artistAlbum:
		if m.artistBrowser.origin == artistFromStatsAlbum || (m.artistBrowser.origin == artistFromSearch && m.artistBrowser.profile.ID == "") {
			m.artistBrowser.open = false
		} else if m.artistBrowser.origin == artistFromPlaylists || m.artistBrowser.origin == artistFromAlbums {
			m.artistBrowser.mode = artistList
		} else {
			m.artistBrowser.mode, m.artistBrowser.tab = artistDetail, 1
		}
	case artistDetail:
		if m.artistBrowser.origin == artistFromLibrary || len(m.artistBrowser.root.items) > 1 {
			m.artistBrowser.mode = artistList
		} else {
			m.artistBrowser.open = false
		}
	default:
		m.artistBrowser.open = false
	}
	if !m.artistBrowser.open {
		m.libraryOpen = false
		if m.artistBrowser.origin == artistFromSearch {
			m.searchOpen = true
			return m.requestSearchArtwork()
		}
		if m.artistBrowser.origin == artistFromStats || m.artistBrowser.origin == artistFromStatsAlbum {
			m.statsOpen = true
			return m.requestStatsArtists()
		}
		return m.requestRecentArtwork()
	}
	return m.requestVisibleArtistImages()
}

func (m model) selectedArtistItem() engine.Track {
	b := m.artistBrowser
	var items []engine.Track
	selection := 0
	switch b.mode {
	case artistList:
		items, selection = b.root.items, b.root.selected
	case artistDetail:
		if b.tab == 0 {
			items, selection = b.songs.items, b.songs.selected
		} else {
			items, selection = b.albums.items, b.albums.selected
		}
	case artistAlbum:
		items, selection = b.albumSongs.items, b.albumSongs.selected
	}
	if selection < 0 || selection >= len(items) {
		return engine.Track{}
	}
	return items[selection]
}

func (m model) activateArtistItem() (tea.Model, tea.Cmd) {
	if m.artistBrowser.busy || m.artistBrowser.err != "" {
		return m, nil
	}
	item := m.selectedArtistItem()
	if item.ID == "" {
		return m, nil
	}
	if item.Kind == "artist" {
		return m.openArtistProfile(item)
	}
	if item.Kind == "album" || item.Kind == "playlist" {
		return m.openArtistAlbum(item)
	}
	if m.audioInitializing() {
		m.artistBrowser.notice = audioInitializingText
		return m, nil
	}
	m.artistBrowser.open, m.libraryOpen, m.statsOpen = false, false, false
	return m.playDiscoveryTrack(item)
}

func (m model) updateArtistBrowser(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	b := &m.artistBrowser
	l := m.artistGeometry()
	switch key.String() {
	case "/":
		m.artistRequestID++
		b.open = false
		m.libraryOpen, m.statsOpen, m.searchOpen, m.sInput, m.sBusy = false, false, true, true, false
		return m, nil
	case "esc", "ctrl+c":
		return m.artistBack()
	case "tab", "shift+tab":
		if b.mode == artistDetail {
			b.tab = 1 - b.tab
			b.notice = ""
			return m.requestVisibleArtistImages()
		}
	case "r", "R":
		m.artistArtRequested = nil
		switch b.mode {
		case artistList:
			if b.origin == artistFromStats {
				return m.openStatsArtist(b.lookupName, true)
			}
			return m.openArtistLibrary(b.root.page)
		case artistDetail:
			return m.openArtistProfile(b.profile)
		case artistAlbum:
			return m.openArtistAlbum(b.album)
		case artistResolving:
			return m.openStatsArtist(b.lookupName, true)
		}
	case "enter":
		return m.activateArtistItem()
	case "e", "E":
		if b.busy || b.err != "" || m.eng == nil {
			return m, nil
		}
		item := m.selectedArtistItem()
		if b.mode == artistAlbum {
			item = b.album
		}
		if item.ID == "" || (item.Kind != "album" && item.Kind != "playlist") {
			return m, nil
		}
		if m.audioInitializing() {
			b.notice = audioInitializingText
			return m, nil
		}
		b.open, m.libraryOpen, m.statsOpen = false, false, false
		return m.playDiscoveryTrack(item)
	case "a", "A":
		item := m.selectedArtistItem()
		if !b.busy && item.ID != "" && (item.Kind == "song" || item.Kind == "album" || item.Kind == "playlist") && m.eng != nil {
			b.notice = "Adding to queue…"
			request, eng, next := m.artistRequestID, m.eng, key.String() == "A"
			return m, func() tea.Msg {
				queue := eng.QueueLater
				notice := "Added to queue: " + artistLabel(item.Title)
				if next {
					queue = eng.QueueNext
					notice = "Playing next: " + artistLabel(item.Title)
				}
				return artistQueueMsg{request: request, notice: notice, err: queue(item.Kind, item.ID)}
			}
		}
	}
	if b.busy {
		return m, nil
	}
	grid := b.mode == artistList || (b.mode == artistDetail && b.tab == 1)
	step := 0
	switch key.String() {
	case "up", "k":
		step = -1
		if grid {
			step = -l.cols
		}
	case "down", "j":
		step = 1
		if grid {
			step = l.cols
		}
	case "left", "h":
		if grid {
			step = -1
		}
	case "right", "l":
		if grid {
			step = 1
		}
	case "pgup", "ctrl+b":
		step = -l.listRows
		if grid {
			step = -l.cols * l.gridRows
		}
	case "pgdown", "ctrl+f":
		step = l.listRows
		if grid {
			step = l.cols * l.gridRows
		}
	}
	if grid {
		g := &b.root
		if b.mode == artistDetail {
			g = &b.albums
		}
		if b.mode == artistList && b.origin != artistFromStats {
			if step > 0 && g.selected >= len(g.items)-1 && len(g.items) == 50 && g.page < 10000 {
				return m.openArtistLibrary(g.page + 50)
			}
			if step < 0 && g.selected == 0 && g.page > 0 {
				return m.openArtistLibrary(max(0, g.page-50))
			}
		}
		g.selected = max(0, min(g.selected+step, len(g.items)-1))
		if key.String() == "home" {
			g.selected = 0
		}
		if key.String() == "end" {
			g.selected = max(0, len(g.items)-1)
		}
		g.start = artistGridStart(*g, l.cols, l.gridRows)
	} else {
		songs := &b.songs
		if b.mode == artistAlbum {
			songs = &b.albumSongs
		}
		songs.selected = max(0, min(songs.selected+step, len(songs.items)-1))
		if key.String() == "home" {
			songs.selected = 0
		}
		if key.String() == "end" {
			songs.selected = max(0, len(songs.items)-1)
		}
		songs.start = statsListStart(songs.selected, songs.start, l.listRows, len(songs.items))
	}
	return m.requestVisibleArtistImages()
}

func (m model) updateArtistMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.artistGeometry()
	b := &m.artistBrowser
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelUp {
		return m.updateArtistBrowser(tea.KeyMsg{Type: tea.KeyUp})
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return m.updateArtistBrowser(tea.KeyMsg{Type: tea.KeyDown})
	}
	if msg.Button != tea.MouseButtonLeft || b.busy || msg.X < l.x || msg.X >= l.x+l.width {
		return m, nil
	}
	if b.mode == artistDetail && msg.Y == l.tabsY {
		if msg.X < l.x+16 {
			b.tab = 0
		} else {
			b.tab = 1
		}
		return m.requestVisibleArtistImages()
	}
	if b.mode == artistList || (b.mode == artistDetail && b.tab == 1) {
		y := msg.Y - l.gridY
		x := msg.X - l.x
		if y < 0 || y >= l.gridRows*(l.cardH+1)-1 || y%(l.cardH+1) == l.cardH || x%(l.cardW+2) >= l.cardW {
			return m, nil
		}
		col, row := x/(l.cardW+2), y/(l.cardH+1)
		if col >= l.cols {
			return m, nil
		}
		index := l.start + row*l.cols + col
		g := &b.root
		if b.mode == artistDetail {
			g = &b.albums
		}
		if index >= len(g.items) {
			return m, nil
		}
		g.selected, g.start = index, l.start
	} else {
		if msg.Y < l.listY || msg.Y >= l.listY+l.listRows {
			return m, nil
		}
		songs := &b.songs
		if b.mode == artistAlbum {
			songs = &b.albumSongs
		}
		start := statsListStart(songs.selected, songs.start, l.listRows, len(songs.items))
		index := start + msg.Y - l.listY
		if index >= len(songs.items) {
			return m, nil
		}
		songs.selected, songs.start = index, start
	}
	return m.activateArtistItem()
}

func (m model) artistGridView(g artistGridState, l artistGeometry) string {
	var rows []string
	for row := 0; row < l.gridRows; row++ {
		var cards []string
		for col := 0; col < l.cols; col++ {
			index := l.start + row*l.cols + col
			if index >= len(g.items) {
				cards = append(cards, strings.Repeat(" ", l.cardW)+strings.Repeat("\n"+strings.Repeat(" ", l.cardW), l.cardH-1))
				continue
			}
			item := g.items[index]
			inner := l.cardW - 2
			pictureCols := 2 * l.thumbRows
			imageRows := m.artistPicture(item, pictureCols, l.thumbRows)
			for i := range imageRows {
				imageRows[i] = pad(strings.Repeat(" ", max(0, (inner-pictureCols)/2))+imageRows[i], inner)
			}
			name := ansi.Truncate(artistLabel(item.Title), inner, "…")
			style := lipgloss.NewStyle().Foreground(fgBright)
			border := borderDim
			if index == g.selected {
				style = style.Foreground(accentHi).Bold(true)
				border = accentHi
			}
			caption := "Artist"
			if item.Kind == "album" {
				caption = "Album"
			} else if item.Kind == "playlist" {
				caption = "Playlist"
			}
			body := strings.Join(imageRows, "\n") + "\n" + pad(style.Render(name), inner) + "\n" + pad(lipgloss.NewStyle().Foreground(fgFaint).Render(caption), inner)
			cards = append(cards, lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Width(inner).Render(body))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, joinArtistCards(cards)...))
	}
	return strings.Join(rows, "\n\n")
}

func joinArtistCards(cards []string) []string {
	result := make([]string, 0, len(cards)*2)
	for i, card := range cards {
		if i > 0 {
			result = append(result, "  ")
		}
		result = append(result, card)
	}
	return result
}

func (m model) artistBrowserView() string {
	l := m.artistGeometry()
	b := m.artistBrowser
	bright := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	fit := func(text string) string { return pad(ansi.Truncate(artistLabel(text), l.width, "…"), l.width) }
	status := func(height int) string {
		text := b.err
		if text == "" {
			text = "Loading…"
		}
		lines := []string{fit(text)}
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", l.width))
		}
		return strings.Join(lines, "\n")
	}
	var content string
	if b.mode == artistList || b.mode == artistResolving {
		title, subtitle := "POMME · ARTISTS", "Your library"
		if b.origin == artistFromPlaylists {
			title = "POMME · PLAYLISTS"
		} else if b.origin == artistFromAlbums {
			title = "POMME · ALBUMS"
		}
		if b.origin == artistFromStats {
			subtitle = "Choose an artist · " + b.lookupName
		} else {
			subtitle += fmt.Sprintf(" · page %d", b.root.page/50+1)
		}
		grid := m.artistGridView(b.root, l)
		if b.busy || b.err != "" || len(b.root.items) == 0 {
			if !b.busy && b.err == "" {
				b.err = "No artists available."
				if b.origin == artistFromPlaylists {
					b.err = "No playlists available."
				} else if b.origin == artistFromAlbums {
					b.err = "No albums available."
				}
			}
			grid = status(l.gridRows*l.cardH + l.gridRows - 1)
		}
		backHints := "/ Search · r Retry · Esc Back"
		if b.origin == artistFromStats {
			backHints = "/ Search · r Retry · Esc Stats"
		}
		hints := "←→/↑↓ Select · Enter/click Open · PgUp/PgDn Page"
		if b.origin == artistFromPlaylists || b.origin == artistFromAlbums {
			backHints = "e Play all · a/A Queue/Next · / Search · r Retry · Esc Back"
		}
		if b.notice != "" {
			hints = b.notice
		}
		content = bright.Render(fit(title)) + "\n" + faint.Render(fit(subtitle)) + "\n" + grid + "\n" + faint.Render(fit(hints)) + "\n" + faint.Render(fit(backHints))
	} else {
		profile := b.profile
		title := "POMME · ARTIST"
		if b.mode == artistAlbum {
			profile = b.album
			title = "POMME · ALBUM"
			if profile.Kind == "playlist" {
				title = "POMME · PLAYLIST"
			}
		}
		pictureCols := 2 * l.heroRows
		photo := strings.Join(m.artistPicture(profile, pictureCols, l.heroRows), "\n")
		infoW := l.width - pictureCols - 3
		info := []string{bright.Render(pad(ansi.Truncate(artistLabel(profile.Title), infoW, "…"), infoW)), ""}
		if b.mode == artistAlbum {
			info = append(info, faint.Render(ansi.Truncate(artistLabel(profile.Artist), infoW, "…")), fmt.Sprintf("%d songs", len(b.albumSongs.items)))
		} else {
			info = append(info, fmt.Sprintf("%d top songs", len(b.songs.items)), fmt.Sprintf("%d albums", len(b.albums.items)))
		}
		for len(info) < l.heroRows {
			info = append(info, "")
		}
		hero := lipgloss.JoinHorizontal(lipgloss.Top, photo, "   ", strings.Join(info, "\n"))
		tabs := "TRACKS"
		if b.mode == artistDetail {
			songs, albums := faint.Render("Top songs"), faint.Render("Albums")
			if b.tab == 0 {
				songs = bright.Render("Top songs")
			} else {
				albums = bright.Render("Albums")
			}
			tabs = songs + "       " + albums
		}
		var body string
		bodyH := l.listRows
		if b.mode == artistDetail && b.tab == 1 {
			body = m.artistGridView(b.albums, l)
			bodyH = l.gridRows*l.cardH + l.gridRows - 1
			if len(b.albums.items) == 0 {
				body = fit("No albums available.") + strings.Repeat("\n"+strings.Repeat(" ", l.width), bodyH-1)
			}
		} else {
			songs := b.songs
			if b.mode == artistAlbum {
				songs = b.albumSongs
			}
			start := statsListStart(songs.selected, songs.start, l.listRows, len(songs.items))
			var lines []string
			for i := start; i < min(start+l.listRows, len(songs.items)); i++ {
				song := songs.items[i]
				prefix := fmt.Sprintf("  %d. ", i+1)
				if i == songs.selected {
					prefix = fmt.Sprintf("› %d. ", i+1)
				}
				duration := ""
				if song.Duration > 0 {
					duration = "  " + fmtTime(song.Duration)
				}
				line := prefix + pad(ansi.Truncate(artistLabel(song.Title), max(0, l.width-lipgloss.Width(prefix)-lipgloss.Width(duration)), "…"), max(0, l.width-lipgloss.Width(prefix)-lipgloss.Width(duration))) + duration
				if i == songs.selected {
					line = lipgloss.NewStyle().Foreground(accentHi).Background(selBg).Render(line)
				}
				lines = append(lines, pad(line, l.width))
			}
			if len(lines) == 0 {
				lines = append(lines, faint.Render(fit("No songs available.")))
			}
			for len(lines) < l.listRows {
				lines = append(lines, strings.Repeat(" ", l.width))
			}
			body = strings.Join(lines, "\n")
		}
		if b.busy || b.err != "" {
			body = status(bodyH)
		}
		hints := "↑↓ Select · Enter/click Play · a/A Queue/Next"
		if b.tab == 1 && b.mode == artistDetail {
			hints = "←→/↑↓ Select · Enter/click Open · e Play album"
		} else if b.mode == artistAlbum {
			hints = "↑↓ Select · Enter/click Play · e Play all · a/A Queue/Next"
		}
		if b.notice != "" {
			hints = b.notice
		}
		backHints := "Tab Songs/Albums · / Search · r Retry · Esc Back"
		if b.mode == artistAlbum {
			backHints = "/ Search · r Retry · Esc Back to artist"
			if b.origin == artistFromPlaylists {
				backHints = "/ Search · r Retry · Esc Playlists"
			} else if b.origin == artistFromAlbums {
				backHints = "/ Search · r Retry · Esc Albums"
			} else if b.origin == artistFromStatsAlbum {
				backHints = "/ Search · r Retry · Esc Stats"
			} else if b.origin == artistFromSearch && b.profile.ID == "" {
				backHints = "/ Search · r Retry · Esc Search"
			}
		}
		content = bright.Render(fit(title)) + "\n" + hero + "\n\n" + pad(tabs, l.width) + "\n\n" + body + "\n" + faint.Render(fit(hints)) + "\n" + faint.Render(fit(backHints))
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, content)
}
