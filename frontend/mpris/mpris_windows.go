//go:build windows

package mpris

// Hardware media buttons are handled by the native backend's SMTC session.
// This bridge registers user-enabled keyboard combinations only.
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type hotkeyConfig struct {
	Enabled   bool   `json:"enabled"`
	PlayPause string `json:"play_pause"`
	Next      string `json:"next"`
	Previous  string `json:"previous"`
}
type binding struct{ modifiers, key uintptr }

func parseBinding(value string) (binding, error) {
	var b binding
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(value)), "+")
	if len(parts) < 2 {
		return b, fmt.Errorf("use modifiers and a key, such as Ctrl+Alt+Space")
	}
	for _, p := range parts[:len(parts)-1] {
		var bit uintptr
		switch strings.TrimSpace(p) {
		case "CTRL":
			bit = 2
		case "ALT":
			bit = 1
		case "SHIFT":
			bit = 4
		default:
			return b, fmt.Errorf("unsupported modifier %q", p)
		}
		if b.modifiers&bit != 0 {
			return b, fmt.Errorf("duplicate modifier")
		}
		b.modifiers |= bit
	}
	if b.modifiers&3 == 0 {
		return b, fmt.Errorf("include Ctrl or Alt")
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	switch key {
	case "SPACE":
		b.key = 0x20
	case "LEFT":
		b.key = 0x25
	case "UP":
		b.key = 0x26
	case "RIGHT":
		b.key = 0x27
	case "DOWN":
		b.key = 0x28
	default:
		if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
			b.key = uintptr(key[0])
		} else if strings.HasPrefix(key, "F") {
			n, e := strconv.Atoi(key[1:])
			if e == nil && n >= 1 && n <= 24 {
				b.key = uintptr(0x70 + n - 1)
			}
		}
	}
	if b.key == 0 {
		return b, fmt.Errorf("unsupported key %q", key)
	}
	return b, nil
}
func loadHotkeys(path string) ([]binding, error) {
	cfg := hotkeyConfig{PlayPause: "Ctrl+Alt+Space", Next: "Ctrl+Alt+Right", Previous: "Ctrl+Alt+Left"}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
		data, _ = json.MarshalIndent(cfg, "", "  ")
		// Exclusive creation never overwrites a user's configuration.
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, e
		}
		_, e = f.Write(append(data, '\n'))
		ce := f.Close()
		if e != nil {
			return nil, e
		}
		return nil, ce
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 8192 {
		return nil, fmt.Errorf("hotkeys.json is too large")
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid hotkeys.json: %w", err)
	}
	if !cfg.Enabled {
		return nil, nil
	}
	var keys []binding
	for _, v := range []string{cfg.PlayPause, cfg.Next, cfg.Previous} {
		b, e := parseBinding(v)
		if e != nil {
			return nil, e
		}
		for _, prior := range keys {
			if b == prior {
				return nil, fmt.Errorf("hotkey combinations must be different")
			}
		}
		keys = append(keys, b)
	}
	return keys, nil
}

var user32 = syscall.NewLazyDLL("user32.dll")
var registerHotKey = user32.NewProc("RegisterHotKey")
var unregisterHotKey = user32.NewProc("UnregisterHotKey")
var peekMessage = user32.NewProc("PeekMessageW")
var getMessage = user32.NewProc("GetMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")
var currentThreadID = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")

type winMessage struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	x, y    int32
	private uint32
}
type Server struct {
	thread uintptr
	done   chan struct{}
	once   sync.Once
}

func Publish(ctl Controls) (*Server, error) {
	dir := os.Getenv("AMTUI_CONFIG_DIR")
	if dir == "" {
		return nil, nil
	}
	keys, err := loadHotkeys(filepath.Join(dir, "hotkeys.json"))
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return startHotkeys(ctl, keys)
}
func startHotkeys(ctl Controls, keys []binding) (*Server, error) {
	s := &Server{done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(s.done)
		s.thread, _, _ = currentThreadID.Call()
		var msg winMessage
		peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
		registered := 0
		defer func() {
			for i := 0; i < registered; i++ {
				unregisterHotKey.Call(0, uintptr(i+1))
			}
		}()
		for i, b := range keys {
			ok, _, e := registerHotKey.Call(0, uintptr(i+1), b.modifiers|0x4000, b.key)
			if ok == 0 {
				ready <- fmt.Errorf("global shortcut %d is unavailable (possibly used by another app): %v", i+1, e)
				return
			}
			registered++
		}
		ready <- nil
		busy := make(chan struct{}, 1)
		for {
			r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			if msg.message != 0x312 {
				continue
			}
			var action func() error
			switch msg.wParam {
			case 1:
				action = ctl.PlayPause
			case 2:
				action = ctl.Next
			case 3:
				action = ctl.Prev
			default:
				continue
			}
			select {
			case busy <- struct{}{}:
				go func() { defer func() { <-busy }(); _ = action() }()
			default:
			}
		}
	}()
	if err := <-ready; err != nil {
		<-s.done
		return nil, err
	}
	return s, nil
}
func (s *Server) Update(State) {}
func (s *Server) Close() {
	if s != nil {
		s.once.Do(func() { postThreadMessage.Call(s.thread, 0x12, 0, 0); <-s.done })
	}
}
