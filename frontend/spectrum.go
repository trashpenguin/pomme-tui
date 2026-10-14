package main

import (
	"github.com/charmbracelet/lipgloss"
	"math"
	"strings"
)

// spectrumPanel uses bounded spectrum history; memory does not grow with playback.
func spectrumPanel(w, h, mode int, bands [32]float64, history [48][32]float64) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	var out strings.Builder
	ramp := []rune(" .:-=+*#%@")
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			band := min(31, x*32/w)
			level := max(0., min(1., bands[band]))
			c := ' '
			switch mode {
			case vizWave:
				if y == h-1-int(level*float64(h-1)) {
					c = '*'
				}
			case vizMirror:
				if math.Abs(float64(y)-float64(h-1)/2) <= level*float64(h)/2 {
					c = '|'
				}
			case vizWaterfall:
				level = max(0., min(1., history[min(47, y*48/h)][band]))
				c = ramp[int(level*float64(len(ramp)-1))]
			}
			out.WriteRune(c)
		}
		if y < h-1 {
			out.WriteByte('\n')
		}
	}
	return lipgloss.NewStyle().Foreground(accentHi).Render(out.String())
}
