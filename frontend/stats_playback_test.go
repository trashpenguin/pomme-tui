package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

// The real Engine launches this test executable as a tiny protocol peer.
// No Apple account, audio device or production profile is used.
func init() {
	if os.Getenv("POMME_STATS_TEST_BACKEND") != "1" {
		return
	}
	decoder, encoder := json.NewDecoder(os.Stdin), json.NewEncoder(os.Stdout)
	fixtures := map[string]json.RawMessage{}
	if path := os.Getenv("POMME_ARTIST_TEST_FIXTURES"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &fixtures) != nil {
			os.Exit(4)
		}
	}
	for {
		var request struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if decoder.Decode(&request) != nil {
			os.Exit(0)
		}
		var result any = map[string]any{}
		if fixture, ok := fixtures[request.Method]; ok {
			result = fixture
		}
		if path := os.Getenv("POMME_ARTIST_RPC_CAPTURE"); path != "" {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				os.Exit(5)
			}
			err = json.NewEncoder(file).Encode(request)
			file.Close()
			if err != nil {
				os.Exit(6)
			}
		}
		if request.Method == "hello" {
			result = map[string]any{"protocol": 1, "authed": true}
		}
		if request.Method == "play" {
			if os.WriteFile(os.Getenv("POMME_STATS_PLAY_CAPTURE"), request.Params, 0600) != nil {
				os.Exit(2)
			}
		}
		if encoder.Encode(map[string]any{"id": request.ID, "result": result}) != nil {
			os.Exit(3)
		}
		if request.Method == "shutdown" {
			os.Exit(0)
		}
	}
}

func TestStatsPlaySendsSelectedCatalogIDToNativeEngine(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(t.TempDir(), "play.json")
	t.Setenv("AMTUI_NATIVE_BACKEND", exe)
	t.Setenv("POMME_STATS_TEST_BACKEND", "1")
	t.Setenv("POMME_STATS_PLAY_CAPTURE", capture)
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	m := interactiveStatsModel(120, 38)
	m.eng = eng
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("no playback command")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("playback failed: %v", msg)
	}
	raw, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var params struct{ Kind, ID string }
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	if params.Kind != "song" || params.ID != "2" {
		t.Fatalf("wrong native play request: %s", raw)
	}
}
