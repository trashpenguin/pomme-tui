package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
)

// The transport hint line ran to 96 cells against a 70-column minimum terminal,
// so its tail was simply cut off — and every key added to the app made that
// worse. The inline hints now carry only what gets a beginner moving, and this
// overlay carries the full set.
//
// Whole sections are packed into responsive columns without clipping entries.

// shortHints is the last thing dropped: whatever else does not fit, the user
// still needs to be told where the full list is.
const shortHints = "? keys "

// transportHints is the one-line reminder in the corner of the transport bar.
func transportHints(focus int) string {
	switch focus {
	case focusLibrary:
		return "↑↓ choose · ↵ open · L selected page · ? keys "
	case focusRecent:
		return "↑↓ recent tracks · ↵ play · / search · ? keys "
	case focusPlayer:
		return "←→ seek · ↑↓ volume · n/p track · v viz · t theme · ? keys "
	}
	return "↑↓ select · ↵ play · / search · space pause · ? keys "
}

// fitHints picks the longest form that fits whole in the space left beside the
// track title. Truncating instead would cut mid-word and, worse, silently.
func fitHints(hints string, room int) string {
	if lipgloss.Width(hints) <= room {
		return hints
	}
	if lipgloss.Width(shortHints) <= room {
		return shortHints
	}
	return ""
}

type helpEntry struct{ keys, action string }

type helpSection struct {
	title   string
	entries []helpEntry
}

// Bindings are grouped by the screen that receives them. Reused letters are
// intentional only across these scopes; query input retains printable text.
var helpSections = []helpSection{
	{"MAIN · PLAYBACK", []helpEntry{
		{"space", "play / pause"}, {"n / p", "next / previous"}, {"s", "shuffle toggle"}, {"r / R", "cycle repeat"},
	}},
	{"MINI PLAYER", []helpEntry{
		{"space", "play / pause"}, {"n / p", "next / previous"}, {"← / →", "seek −5 / +5 sec"}, {"↑ / ↓", "volume up / down"}, {"m", "expand player"}, {"T", "cycle sleep timer"}, {"/", "search Apple Music"}, {"q / ctrl+c", "quit player"},
	}},
	{"MAIN · PLAYER FOCUS", []helpEntry{
		{"← / →", "seek −5 / +5 sec"}, {"↑ / ↓", "volume up / down"},
	}},
	{"MAIN · NAVIGATION", []helpEntry{
		{"tab / shift+tab", "next / previous panel"}, {"↑↓ / j k", "move selection"}, {"enter", "play / open selected"}, {"L", "open Library page"}, {"/", "search Apple Music"}, {"PgUp/Dn", "page selection"},
	}},
	{"MAIN · QUEUE FOCUS", []helpEntry{
		{"delete", "remove selected track"}, {"ctrl+↑/↓", "move selected track"}, {"ctrl+x", "clear queue"},
	}},
	{"MAIN · SESSION", []helpEntry{
		{"a", "account menu"}, {"S twice", "save queue as playlist"}, {"F5", "reload engine"}, {"q / ctrl+c", "quit player"},
	}},
	{"MAIN · APPEARANCE", []helpEntry{
		{"v", "cycle visualizer"}, {"t", "cycle theme"}, {"m", "toggle mini player"}, {"T", "cycle sleep timer"},
	}},
	{"SEARCH · QUERY INPUT", []helpEntry{
		{"type / backspace", "edit query"}, {"enter", "search / retry"}, {"tab / shift+tab", "next / previous type"}, {"↓ / ↑", "browse results"}, {"ctrl+p/n", "query history"}, {"F1", "help; ? stays text"}, {"esc / ctrl+c", "close search"},
	}},
	{"SEARCH · RESULTS", []helpEntry{
		{"↑↓ / j k", "select result"}, {"enter / dblclick", "play song / open item"}, {"e / E", "play selected item"}, {"a / A", "queue / play next"}, {"tab / shift+tab", "next / previous type"}, {"Home/End", "first / last result"}, {"PgUp/Dn", "page results"}, {"/", "edit query"}, {"L", "open matching Library"}, {"esc / ctrl+c", "close search"},
	}},
	{"LIBRARY · SONG LISTS", []helpEntry{
		{"↑↓ / j k", "select song"}, {"enter", "play selected song"}, {"a / A", "queue / play next"}, {"PgUp/Dn", "page songs"}, {"/", "catalog search"}, {"esc / ctrl+c", "back to main"},
	}},
	{"ARTISTS / COLLECTIONS", []helpEntry{
		{"↑↓←→ / h j k l", "select card / song"}, {"enter / click", "open item / play song"}, {"e / E", "play album / playlist"}, {"a / A", "queue / play next"}, {"tab / shift+tab", "artist songs / albums"}, {"Home/End", "first / last item"}, {"PgUp/Dn", "page items"}, {"r / R", "retry this page"}, {"/", "catalog search"}, {"esc / ctrl+c", "back"},
	}},
	{"STATS", []helpEntry{
		{"tab / shift+tab", "track / artist / album"}, {"↑↓ / j k", "select ranking"}, {"enter", "play track / open item"}, {"e / E", "play selected album"}, {"←→ / h l", "change time range"}, {"Home/End", "first / last ranking"}, {"r / R", "refresh history"}, {"esc / ctrl+c", "back to main"},
	}},
	{"DISCOVERY", []helpEntry{
		{"↑↓ / j k", "select recommendation"}, {"enter", "open / play song"}, {"e / E", "play selection"}, {"a / A", "queue / play next"}, {"w / W", "confirm save to Library"}, {"P (in details)", "queue whole collection"}, {"q", "open queue manager"}, {"PgUp/Dn", "page recommendations"}, {"esc / ctrl+c", "back"},
	}},
	{"SAVE CONFIRMATION", []helpEntry{
		{"y / Y / enter", "confirm Library save"}, {"n / N / esc", "cancel save"},
	}},
	{"QUEUE MANAGER", []helpEntry{
		{"↑↓ / j k", "select queue track"}, {"enter", "play selected track"}, {"del / backspace", "remove selected track"}, {"ctrl+↑/↓", "move selected track"}, {"ctrl+x", "clear queue"}, {"PgUp/Dn", "page queue"}, {"esc / q / ctrl+c", "back"},
	}},
	{"ACCOUNT", []helpEntry{
		{"enter", "sign out"}, {"esc / other key", "close account"},
	}},
	{"HELP", []helpEntry{
		{"F1 / ?", "toggle help"}, {"←→ / PgUp/Dn", "previous / next page"}, {"ctrl+b/f", "previous / next page"}, {"Home/End", "first / last page"}, {"mouse wheel", "previous / next page"}, {"esc / ctrl+c", "close help"},
	}},
}

type helpPageLayout struct{ columns [][]helpSection }
type helpLayout struct {
	pages                                 []helpPageLayout
	rows, keyWidth, colWidth, cols, width int
}

// Pack complete sections rather than slicing rendered lines across pages.
// Narrow terminals use one column so key names and descriptions stay whole.
func (m model) helpLayout() helpLayout {
	l := helpLayout{rows: max(1, m.h-6), cols: 1}
	actionWidth := 0
	for _, section := range helpSections {
		for _, entry := range section.entries {
			l.keyWidth = max(l.keyWidth, lipgloss.Width(entry.keys))
			actionWidth = max(actionWidth, lipgloss.Width(entry.action))
		}
	}
	l.colWidth = 2 + l.keyWidth + 2 + actionWidth
	for _, section := range helpSections {
		l.colWidth = max(l.colWidth, lipgloss.Width(section.title)+2)
	}
	if 2*l.colWidth+3 <= m.w-10 {
		l.cols = 2
	}
	l.width = l.cols*l.colWidth + 3*(l.cols-1)
	page := helpPageLayout{columns: make([][]helpSection, l.cols)}
	col, used := 0, 0
	for _, section := range helpSections {
		// Every current section fits the minimum height. Repeat its heading if a
		// future section grows beyond one page instead of leaving orphaned keys.
		chunks := []helpSection{}
		for start := 0; start < len(section.entries); {
			end := min(start+max(1, l.rows-1), len(section.entries))
			part := section
			part.entries = section.entries[start:end]
			chunks = append(chunks, part)
			start = end
		}
		for _, part := range chunks {
			cost := len(part.entries) + 1
			if used > 0 {
				cost++
			}
			if used+cost > l.rows {
				col++
				used = 0
				cost = len(part.entries) + 1
			}
			if col >= l.cols {
				l.pages = append(l.pages, page)
				page = helpPageLayout{columns: make([][]helpSection, l.cols)}
				col = 0
			}
			page.columns[col] = append(page.columns[col], part)
			used += cost
		}
	}
	l.pages = append(l.pages, page)
	return l
}
func (m model) helpPageCount() int { return len(m.helpLayout().pages) }
func (m model) helpSectionTitle() string {
	switch {
	case m.acctOpen:
		return "ACCOUNT"
	case m.artistBrowser.open:
		return "ARTISTS / COLLECTIONS"
	case m.libraryOpen:
		return "LIBRARY · SONG LISTS"
	case m.queueOpen:
		return "QUEUE MANAGER"
	case m.discoveryOpen && m.saveConfirmOpen:
		return "SAVE CONFIRMATION"
	case m.discoveryOpen:
		return "DISCOVERY"
	case m.statsOpen:
		return "STATS"
	case m.searchOpen && m.sInput:
		return "SEARCH · QUERY INPUT"
	case m.searchOpen:
		return "SEARCH · RESULTS"
	case m.mini:
		return "MINI PLAYER"
	case m.focus == focusQueue:
		return "MAIN · QUEUE FOCUS"
	case m.focus == focusPlayer:
		return "MAIN · PLAYER FOCUS"
	default:
		return "MAIN · NAVIGATION"
	}
}
func (m model) openHelp() (tea.Model, tea.Cmd) {
	m.helpOpen = true
	m.helpPage = 0
	target := m.helpSectionTitle()
	for i, page := range m.helpLayout().pages {
		for _, col := range page.columns {
			for _, section := range col {
				if section.title == target {
					m.helpPage = i
					return m, nil
				}
			}
		}
	}
	return m, nil
}
func (m model) updateHelp(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case pageDirection(key) > 0 || key.String() == "right" || key.String() == "down" || key.String() == "tab":
		m.helpPage = min(m.helpPage+1, m.helpPageCount()-1)
	case pageDirection(key) < 0 || key.String() == "left" || key.String() == "up" || key.String() == "shift+tab":
		m.helpPage = max(0, m.helpPage-1)
	case key.String() == "home":
		m.helpPage = 0
	case key.String() == "end":
		m.helpPage = m.helpPageCount() - 1
	case key.String() == "esc" || key.String() == "ctrl+c" || key.String() == "?" || key.String() == "f1":
		m.helpOpen = false
	}
	return m, nil
}
func renderHelpColumn(sections []helpSection, keyWidth, width int) []string {
	accentStyle := lipgloss.NewStyle().Foreground(accentHi)
	textStyle := lipgloss.NewStyle().Foreground(fgBright)
	var lines []string
	for i, section := range sections {
		if i > 0 {
			lines = append(lines, pad("", width))
		}
		lines = append(lines, pad(" "+accentStyle.Bold(true).Render(section.title), width))
		for _, entry := range section.entries {
			lines = append(lines, pad("  "+accentStyle.Render(pad(entry.keys, keyWidth))+"  "+textStyle.Render(entry.action), width))
		}
	}
	return lines
}
func (m model) helpView() string {
	l := m.helpLayout()
	page := min(max(0, m.helpPage), len(l.pages)-1)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	bright := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	var columns [][]string
	for _, sections := range l.pages[page].columns {
		lines := renderHelpColumn(sections, l.keyWidth, l.colWidth)
		for len(lines) < l.rows {
			lines = append(lines, pad("", l.colWidth))
		}
		columns = append(columns, lines)
	}
	var body []string
	for row := 0; row < l.rows; row++ {
		line := columns[0][row]
		if l.cols == 2 {
			line += faint.Render(" │ ") + columns[1][row]
		}
		body = append(body, line)
	}
	footer := fmt.Sprintf(" %d/%d · ←/→ PgUp/Dn · Esc/?/F1 close", page+1, len(l.pages))
	content := strings.Join(body, "\n") + "\n\n" + pad(faint.Render(footer), l.width)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Width(l.width).Render(content)
	subtitle := faint.Render("Keys depend on screen · F1 help · letter case matters")
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, bright.Render("POMME · CONTROLS")+"\n"+subtitle+"\n"+box)
}
