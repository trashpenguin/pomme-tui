package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"
)

func (m model) miniState() string {
	switch {
	case m.st.Recovering:
		return "RECONNECTING"
	case m.audioInitializing():
		return "INITIALIZING"
	case m.initFailed:
		return "PLAYBACK ERROR"
	case m.loading != "":
		return "LOADING"
	case m.st.Now.Title == "":
		return "READY"
	case m.st.Playing:
		return "PLAYING"
	default:
		return "PAUSED"
	}
}
func miniFit(text string, width int) string {
	return pad(ansi.Truncate(artistLabel(text), max(0, width), "…"), max(0, width))
}
func miniPair(left, right string, width int) string {
	if lipgloss.Width(left)+lipgloss.Width(right)+2 > width {
		return pad(ansi.Truncate(left, width, "…"), width)
	}
	return left + strings.Repeat(" ", width-lipgloss.Width(left)-lipgloss.Width(right)) + right
}
func miniHints(width int) string {
	for _, hint := range []string{
		"Space pause · n/p skip · ←→ seek · ↑↓ vol · m expand · F1 keys · q quit",
		"Space pause · n/p skip · m expand · T timer · F1 keys · q quit",
		"Space pause · n/p skip · m expand · F1 keys",
		"m expand · Space pause · F1 keys",
		"m expand · F1 keys", "m expand", "m",
	} {
		if lipgloss.Width(hint) <= width {
			return hint
		}
	}
	return ""
}
func (m model) miniProgress(width int) string {
	pos, dur := max(time.Duration(0), m.st.Pos), max(time.Duration(0), m.st.Dur)
	if dur > 0 {
		pos = min(pos, dur)
	}
	elapsed, total := fmtTime(pos), fmtTime(dur)
	barWidth := width - lipgloss.Width(elapsed) - lipgloss.Width(total) - 2
	if barWidth < 1 {
		return miniFit(elapsed+" / "+total, width)
	}
	filled := 0
	if dur > 0 {
		filled = int(float64(pos) / float64(dur) * float64(barWidth))
	}
	dim := lipgloss.NewStyle().Foreground(fgDim)
	return dim.Render(elapsed) + " " + m.progressBar(barWidth, filled) + " " + dim.Render(total)
}
func (m model) miniView() string {
	if m.w < 1 || m.h < 1 {
		return ""
	}
	bright := lipgloss.NewStyle().Foreground(fgBright).Bold(true)
	accentStyle := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	title, artist := m.st.Now.Title, m.st.Now.Artist
	if title == "" {
		title, artist = "Nothing playing", "Press / to search Apple Music"
	}
	if m.loading != "" && m.loading != m.st.Now.Title {
		title, artist = m.loading, "Preparing playback…"
	}
	if m.st.Now.Album != "" && m.w >= 68 && m.loading == "" {
		artist += " · " + m.st.Now.Album
	}
	state := m.miniState()
	if m.w < 36 || m.h < 8 {
		width := m.w
		detail := artist
		if m.note != "" {
			detail = m.note
		}
		lines := []string{accentStyle.Render(miniFit("[>_] POMME · "+state, width)), bright.Render(miniFit(title, width)), dim.Render(miniFit(detail, width)), m.miniProgress(width), faint.Render(miniFit(miniHints(width), width))}
		lines = lines[:min(len(lines), m.h)]
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
	}
	inner := min(86, m.w-4)
	contentWidth := inner - 2
	headerRows := 2
	if m.h >= 10 {
		headerRows = 3
	}
	infoWidth := contentWidth
	timer := ""
	if !m.sleepUntil.IsZero() {
		timer = "sleep " + fmtTime(max(time.Duration(0), time.Until(m.sleepUntil)))
	}
	meta := fmt.Sprintf("VOL %d%%", max(0, min(m.st.Volume, 100)))
	if timer != "" {
		if lipgloss.Width(state)+lipgloss.Width(meta)+lipgloss.Width(timer)+5 <= infoWidth {
			meta += " · " + timer
		} else {
			meta = timer
		}
	}
	shuffle := "off"
	if m.st.Shuffle {
		shuffle = "on"
	}
	repeat := []string{"off", "one", "all"}[max(0, min(m.st.Repeat, 2))]
	info := []string{miniPair(accentStyle.Render("POMME"), faint.Render("MINI PLAYER"), infoWidth), miniPair(accentStyle.Render(state), dim.Render(meta), infoWidth)}
	if headerRows >= 3 {
		info = append(info, faint.Render(miniFit("⇄ "+shuffle+"   ↻ "+repeat, infoWidth)))
	}

	var lines []string
	for i := 0; i < headerRows; i++ {
		lines = append(lines, pad(info[i], infoWidth))
	}
	lines = append(lines, bright.Render(miniFit(title, contentWidth)))
	detail := artist
	hasNoticeRow := m.h >= headerRows+7
	if !hasNoticeRow && m.note != "" {
		detail = m.note
	}
	lines = append(lines, dim.Render(miniFit(detail, contentWidth)), m.miniProgress(contentWidth))
	if hasNoticeRow {
		lines = append(lines, dim.Render(miniFit(m.note, contentWidth)))
	}
	lines = append(lines, faint.Render(miniFit(miniHints(contentWidth), contentWidth)))
	for i := range lines {
		lines[i] = " " + pad(lines[i], contentWidth) + " "
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Width(inner).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}
