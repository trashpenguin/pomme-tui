package main

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestExtraVisualizerBounds(t *testing.T) {
	for _, mode := range []int{vizWave, vizMirror, vizWaterfall} {
		for _, size := range [][2]int{{1, 1}, {17, 9}, {80, 24}} {
			output := spectrumPanel(size[0], size[1], mode, fullBands(0.8), [48][32]float64{})
			rows := strings.Split(output, "\n")
			if len(rows) != size[1] {
				t.Fatalf("mode %d height %d", mode, len(rows))
			}
			for _, row := range rows {
				if lipgloss.Width(row) != size[0] {
					t.Fatalf("mode %d width %d", mode, lipgloss.Width(row))
				}
			}
		}
	}
}

func TestProgressClampsToTrack(t *testing.T) {
	m := model{}
	for _, filled := range []int{-10, 0, 5, 10, 100} {
		if got := lipgloss.Width(m.progressBar(10, filled)); got != 10 {
			t.Fatalf("filled %d width %d", filled, got)
		}
	}
	if m.progressBar(-1, 5) != "" {
		t.Fatal("negative width must be empty")
	}
}
