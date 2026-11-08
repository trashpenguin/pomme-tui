package main

import (
	"context"
	"image"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type artistImageMsg struct {
	key string
	img image.Image
}

func artistLabel(text string) string { return strings.Join(strings.Fields(ansi.Strip(text)), " ") }
func artistImageKey(item engine.Track) string {
	if item.Art == "" {
		return ""
	}
	return item.Kind + ":" + item.ID + ":" + item.Art
}

func (m model) requestArtistImages(items []engine.Track) (model, tea.Cmd) {
	if !configBool(m.cfg, "artwork.enabled", true) {
		return m, nil
	}
	if m.artistArt == nil {
		m.artistArt = newArtCache(16)
	}
	if m.artistArtRequested == nil || len(m.artistArtRequested) > 256 {
		m.artistArtRequested = map[string]bool{}
	}
	var commands []tea.Cmd
	for _, item := range items {
		key := artistImageKey(item)
		if key == "" || m.artistArtRequested[key] {
			continue
		}
		if _, ok := m.artistArt.get(key); ok {
			continue
		}
		m.artistArtRequested[key] = true
		template := item.Art
		commands = append(commands, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
			defer cancel()
			img, err := fetchArtwork(ctx, artworkURL(template, 96))
			if err != nil {
				return artistImageMsg{key: key}
			}
			// Retain only small thumbnails, even if a server ignored the requested size.
			if b := img.Bounds(); b.Dx() > 96 || b.Dy() > 96 {
				small := image.NewRGBA(image.Rect(0, 0, 96, 96))
				for y := 0; y < 96; y++ {
					for x := 0; x < 96; x++ {
						small.Set(x, y, img.At(b.Min.X+x*b.Dx()/96, b.Min.Y+y*b.Dy()/96))
					}
				}
				img = small
			}
			return artistImageMsg{key: key, img: img}
		})
	}
	return m, tea.Batch(commands...)
}

func (m model) artistPicture(item engine.Track, cols, rows int) []string {
	if img, ok := m.artistArt.get(artistImageKey(item)); ok && img != nil && configBool(m.cfg, "artwork.enabled", true) {
		return renderArtwork(img, cols, rows)
	}
	initials := ""
	for _, word := range strings.Fields(artistLabel(item.Title)) {
		if len([]rune(initials)) >= 2 {
			break
		}
		initials += string([]rune(word)[0])
	}
	if initials == "" {
		initials = "♪"
	}
	label := lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render(strings.ToUpper(initials))
	result := make([]string, rows)
	for i := range result {
		result[i] = strings.Repeat(" ", cols)
	}
	if rows > 0 {
		result[rows/2] = pad(strings.Repeat(" ", max(0, (cols-lipgloss.Width(label))/2))+label, cols)
	}
	return result
}
