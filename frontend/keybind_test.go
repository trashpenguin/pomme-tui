package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRepeatKeysPreserveEngineAndInitializationState(t *testing.T) {
	for _, key := range []string{"r", "R"} {
		t.Run(key, func(t *testing.T) {
			m := model{phase: phaseReady, note: "keep", lyFor: "current-song", initFailed: true, initStart: 10}
			next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			got := next.(model)
			if cmd == nil {
				t.Fatal("repeat key did not dispatch a command")
			}
			if got.note != m.note || got.lyFor != m.lyFor || got.initFailed != m.initFailed || got.initStart != m.initStart {
				t.Fatal("repeat key reloaded the engine or reset initialization")
			}
		})
	}
}

func TestF5ReloadsEngine(t *testing.T) {
	m := model{phase: phaseReady, lyFor: "current-song", initFailed: true, initStart: 10}
	next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyF5})
	got := next.(model)
	if cmd == nil || got.note != "reloading engine…" || got.lyFor != "" || got.initFailed || got.initStart != 0 {
		t.Fatal("F5 did not dispatch reload and reset initialization")
	}
}

func TestFailedConnectionRetriesOnlyWithF5(t *testing.T) {
	m := model{phase: phaseFail, statusCh: make(chan string, 1)}
	for _, key := range []string{"r", "R"} {
		next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd != nil || next.(model).phase != phaseFail {
			t.Fatalf("%q unexpectedly retried the connection", key)
		}
	}
	next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyF5})
	if cmd == nil || next.(model).phase != phaseBoot {
		t.Fatal("F5 did not retry the failed connection")
	}
}
