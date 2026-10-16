package main

import (
	"strings"

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
	case "ctrl+l":
		m.libraryCategory = (m.libraryCategory + 1) % 5
		m.libraryOffset = 0
		m.sBusy = true
		return m, m.pagedLibraryCmd()
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
			return m, doCmd(func() error { return eng.Play(kind, id) })
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
