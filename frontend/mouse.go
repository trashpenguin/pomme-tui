package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Clicking only works if the hit-test agrees with the renderer down to the
// cell, so the panel arithmetic lives here once and View reads it rather than
// recomputing it. Same for the progress bar: transportPanel takes its width
// from progressGeometry, so a click cannot land where the bar is not drawn.

type layout struct {
	topH   int // rows above the transport
	leftW  int // full width of the queue column, borders included
	rightW int
	qw, qh int // queue panel content size
	libH   int // full height of the library box, 0 when hidden
	recH   int // full height of the recently-played box, 0 when hidden
	vw, vh int // visualizer panel
	lh     int // lyrics panel content height
}

func (m model) layout() layout {
	l := layout{topH: m.h - 7}
	l.leftW = m.w * 42 / 100
	l.rightW = m.w - l.leftW
	l.qw = l.leftW - 2
	if l.topH >= 15 {
		l.libH = 10 // seven library rows plus the panel title and borders
	}
	if l.topH >= 24 {
		l.recH = 9 // larger square covers, with two recent tracks visible
	}
	if l.topH-2-l.libH-l.recH < 4 {
		l.recH = 0 // keep a useful queue visible on shorter terminals
	}
	if l.topH-2-l.libH < 3 {
		l.libH = 0
	}
	l.qh = l.topH - 2 - l.libH - l.recH
	l.vh = l.topH * 62 / 100
	l.vw = l.rightW - 2
	l.lh = l.topH - l.vh - 2
	return l
}

// queueRows is how many tracks the queue panel shows at once, and queueTop the
// screen row the first of them is drawn on. panel() spends one row on the
// border and one on its title before the body starts.
func (l layout) queueRows() int { return l.qh - 1 }
func (l layout) queueTop() int  { return 5 }

// libraryTop and recentTop are the first body rows below each panel title.
func (l layout) libraryTop() int { return l.qh + 2 + 2 + 3 }
func (l layout) recentTop() int  { return l.qh + l.libH + 2 + 2 + 3 }

// progressGeometry returns the column the progress bar starts at within the
// transport's content, and its width. transportPanel lays the second line out
// as " ▄▄ " then the play icon then the elapsed time, which is six cells plus
// the clock before the bar begins.
func (m model) progressGeometry(w int) (x, width int) {
	times := fmtTime(m.st.Pos) + " "
	timee := " " + fmtTime(m.st.Dur)
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	right := fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep) +
		strings.Repeat("▮", 6) + fmt.Sprintf(" %d%% ", m.st.Volume)
	barW := max(10, w-lipgloss.Width(times)-lipgloss.Width(timee)-lipgloss.Width(right)-8)
	return 6 + lipgloss.Width(times), barW
}

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseReady || m.helpOpen || m.acctOpen {
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress && msg.Y < 3 {
		m.searchOpen, m.sInput = true, true
		return m, nil
	}
	if m.searchOpen || m.libraryOpen || m.discoveryOpen || m.statsOpen {
		return m, nil
	}
	lay := m.layout()
	inQueue := msg.X < lay.leftW && msg.Y >= lay.queueTop() &&
		msg.Y < lay.queueTop()+lay.queueRows()
	libraryPanelTop := lay.queueTop() + lay.qh
	inLibrary := lay.libH > 0 && msg.X < lay.leftW &&
		msg.Y >= libraryPanelTop && msg.Y < libraryPanelTop+lay.libH
	recentPanelTop := libraryPanelTop + lay.libH
	inRecent := lay.recH > 0 && msg.X < lay.leftW &&
		msg.Y >= recentPanelTop && msg.Y < recentPanelTop+lay.recH

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		switch {
		case inQueue:
			m.focus = focusQueue
			m.selIdx = max(0, min(m.selIdx+step, len(m.st.Queue)-1))
		case inLibrary:
			m.focus = focusLibrary
			m.librarySelection = max(0, min(m.librarySelection+step, 6))
		case inRecent:
			m.focus = focusRecent
			m.recentSel = max(0, min(m.recentSel+step, len(m.recent)-1))
			return m.requestRecentArtwork()
		}
		return m, nil

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		return m.click(msg, lay, inQueue, inLibrary, inRecent)
	}
	return m, nil
}

func (m model) click(msg tea.MouseMsg, lay layout, inQueue, inLibrary, inRecent bool) (tea.Model, tea.Cmd) {
	eng := m.eng
	switch {
	case inQueue && len(m.st.Queue) > 0:
		// Same windowing queuePanel uses, so the row under the pointer is the
		// track that was drawn there.
		rows := lay.queueRows()
		start := max(0, min(m.selIdx-rows/2, len(m.st.Queue)-rows))
		i := start + msg.Y - lay.queueTop()
		if i < 0 || i >= len(m.st.Queue) {
			return m, nil
		}
		m.focus, m.selIdx = focusQueue, i
		if m.audioInitializing() {
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
		m.loading = m.st.Queue[i].Title
		m.loadSnap, m.loadStart, m.loadHideQueue = snap(m.st), m.t, false
		return m, doCmd(func() error { return eng.JumpTo(i) })

	case inLibrary:
		// Ignore clicks on the border and title; category rows begin below them.
		i := msg.Y - lay.libraryTop()
		if i < 0 || i >= 7 {
			return m, nil
		}
		m.focus, m.librarySelection = focusLibrary, i
		return m.openLibrarySelection(i)

	case inRecent && len(m.recent) > 0:
		// Recently played rows begin below their border and title.
		row := msg.Y - lay.recentTop()
		if row < 0 {
			return m, nil
		}
		start, count := m.recentWindow(lay.recH - 3)
		card := row / recentCardRows
		if card >= count {
			return m, nil
		}
		i := start + card
		m.focus, m.recentSel = focusRecent, i
		if m.audioInitializing() {
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
		tr := m.recent[i]
		m.loading, m.loadSnap, m.loadStart = tr.Title, snap(m.st), m.t
		m.loadHideQueue = true
		m.beginAudioInitialization()
		kind, id := tr.Kind, tr.ID
		play := doCmd(func() error { return eng.Play(kind, id) })
		m, art := m.requestRecentArtwork()
		return m, tea.Batch(play, art)

	case msg.Y == m.h-2 && m.st.Dur > 0:
		// The transport's second line. Seeking is the one thing a pointer does
		// better than a keyboard, so it is worth the geometry.
		barX, barW := m.progressGeometry(m.w - 2)
		x := msg.X - 1 - barX
		if x < 0 || x >= barW {
			return m, nil
		}
		m.focus = focusPlayer
		m.st.Pos = time.Duration(float64(m.st.Dur) * float64(x) / float64(barW))
		m.wv.reset() // the recorded waveform no longer matches the new position
		pos := m.st.Pos
		return m, doCmd(func() error { return eng.SeekTo(pos) })
	}
	return m, nil
}
