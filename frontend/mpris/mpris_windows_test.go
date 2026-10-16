//go:build windows

package mpris

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHotkeysDefaultOff(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AMTUI_CONFIG_DIR", dir)
	s, e := Publish(nil)
	if e != nil || s != nil {
		t.Fatalf("default registered shortcuts: %v %v", s, e)
	}
	data, e := os.ReadFile(filepath.Join(dir, "hotkeys.json"))
	if e != nil {
		t.Fatal(e)
	}
	if string(data) == "" {
		t.Fatal("missing editable configuration")
	}
	s, e = Publish(nil)
	if e != nil || s != nil {
		t.Fatal("saved defaults enabled shortcuts")
	}
}
func TestHotkeyValidation(t *testing.T) {
	for _, v := range []string{"Space", "Shift+A", "Ctrl+Ctrl+A", "Ctrl+Alt+F25", "Ctrl+Alt+Unknown"} {
		if _, e := parseBinding(v); e == nil {
			t.Errorf("accepted %s", v)
		}
	}
	for _, v := range []string{"Ctrl+Alt+Space", "Ctrl+Shift+F12", "Alt+Right", " ctrl + alt + k "} {
		if _, e := parseBinding(v); e != nil {
			t.Errorf("rejected %s: %v", v, e)
		}
	}
	path := filepath.Join(t.TempDir(), "hotkeys.json")
	os.WriteFile(path, []byte(`{"enabled":true,"next":"Ctrl+Alt+Space"}`), 0600)
	if _, e := loadHotkeys(path); e == nil {
		t.Fatal("duplicate bindings accepted")
	}
	os.WriteFile(path, []byte(`{"enabled":false,"next":"invalid"}`), 0600)
	if b, e := loadHotkeys(path); e != nil || len(b) != 0 {
		t.Fatal("disabled settings attempted registration")
	}
}

type testControls struct{ events chan string }

func (c testControls) PlayPause() error           { c.events <- "play"; return nil }
func (c testControls) Next() error                { c.events <- "next"; return nil }
func (c testControls) Prev() error                { c.events <- "previous"; return nil }
func (c testControls) SeekTo(time.Duration) error { return nil }
func (c testControls) SetVolume(int) error        { return nil }
func (c testControls) Quit()                      {}
func TestWindowsHotkeyRoutingAndRelease(t *testing.T) {
	ctl := testControls{make(chan string, 3)}
	keys := []binding{{7, 0x83}, {7, 0x84}, {7, 0x85}} // Ctrl+Alt+Shift+F20/F21/F22
	s, e := startHotkeys(ctl, keys)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for i, want := range []string{"play", "next", "previous"} {
		postThreadMessage.Call(s.thread, 0x312, uintptr(i+1), 0)
		select {
		case got := <-ctl.events:
			if got != want {
				t.Fatal(got)
			}
		case <-time.After(time.Second):
			t.Fatal("hotkey event timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if other, e := startHotkeys(ctl, keys); e == nil {
		other.Close()
		t.Fatal("conflicting registration succeeded")
	}
	s.Close()
	s.Close()
	other, e := startHotkeys(ctl, keys)
	if e != nil {
		t.Fatalf("hotkeys not released: %v", e)
	}
	other.Close()
}
