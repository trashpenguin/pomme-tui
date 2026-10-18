package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	rpc    *rpcClient
	cmd    *exec.Cmd
	waited chan struct{}
	once   sync.Once
}

func backendPath() (string, error) {
	if path := os.Getenv("AMTUI_NATIVE_BACKEND"); path != "" {
		return filepath.Abs(path)
	}
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "apple-native-engine.exe"), nil
}
func start(path string) (*Engine, error) {
	cmd := nativeCommand(path)
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	writer, err := cmd.StdinPipe()
	if err != nil {
		reader.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	engine := &Engine{cmd: cmd, rpc: newRPC(reader, writer), waited: make(chan struct{})}
	go func() { _ = cmd.Wait(); engine.rpc.close(); close(engine.waited) }()
	return engine, nil
}

// Connect restores native auth and opens the temporary login helper only if needed.
func Connect(status func(string)) (*Engine, error) {
	return ConnectContext(context.Background(), status)
}
func ConnectContext(parent context.Context, status func(string)) (*Engine, error) {
	path, err := backendPath()
	if err != nil {
		return nil, err
	}
	status("starting native Apple Music…")
	for attempt := 0; attempt < 2; attempt++ {
		if err := parent.Err(); err != nil {
			return nil, err
		}
		engine, err := start(path)
		if err != nil {
			return nil, err
		}
		var hello struct {
			Protocol int  `json:"protocol"`
			Authed   bool `json:"authed"`
		}
		ctx, cancel := context.WithTimeout(parent, 60*time.Second)
		err = engine.rpc.call(ctx, "hello", nil, &hello)
		cancel()
		if err == nil && hello.Protocol != 1 {
			engine.Close()
			return nil, errors.New("unsupported native protocol version")
		}
		if err == nil && hello.Authed {
			status("connected · native playback")
			return engine, nil
		}
		engine.Close()
		if err := parent.Err(); err != nil {
			return nil, err
		}
		needsLogin := err == nil || strings.Contains(err.Error(), "sign in again")
		if !needsLogin || attempt == 1 {
			if err != nil {
				return nil, err
			}
			return nil, errors.New("Apple Music sign-in required")
		}
		status("sign in through the Apple Music window…")
		login := nativeCommandContext(parent, path, "login", "--owned-stdin")
		login.Stderr = io.Discard
		owner, pipeErr := login.StdinPipe()
		if pipeErr != nil {
			return nil, pipeErr
		}
		if err := login.Run(); err != nil {
			owner.Close()
			return nil, errors.New("Apple Music sign-in did not complete")
		}
		owner.Close()
	}
	return nil, errors.New("Apple Music connection failed")
}

func (e *Engine) Close() {
	if e == nil {
		return
	}
	e.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = e.rpc.call(ctx, "shutdown", nil, nil)
		cancel()
		e.rpc.close()
		select {
		case <-e.waited:
		case <-time.After(3 * time.Second):
			_ = e.cmd.Process.Kill()
			<-e.waited
		}
	})
}
func (e *Engine) call(method string, params any, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), methodTimeout(method))
	defer cancel()
	return e.rpc.call(ctx, method, params, out)
}

func methodTimeout(method string) time.Duration {
	switch method {
	case "save_playlist":
		return 310 * time.Second
	case "search", "library", "library_page", "artist_page", "play", "queue_next", "queue_later", "save_item":
		return 35 * time.Second
	default:
		return 5 * time.Second
	}
}
func (e *Engine) State() (State, error) {
	var raw struct {
		Authed, Playing, Initializing bool
		PosMs, DurMs                  uint64
		Volume                        int
		Shuffle                       bool
		Repeat                        int
		Now                           *wireTrack
		Queue                         []wireTrack
		QueuePos                      int
		Err                           string
	}
	if err := e.call("state", nil, &raw); err != nil {
		return State{}, err
	}
	state := State{Authed: raw.Authed, Playing: raw.Playing, Initializing: raw.Initializing, Pos: time.Duration(raw.PosMs) * time.Millisecond, Dur: time.Duration(raw.DurMs) * time.Millisecond,
		Volume: raw.Volume, Shuffle: raw.Shuffle, Repeat: raw.Repeat, Queue: tracks(raw.Queue), QueuePos: raw.QueuePos, Err: raw.Err}
	if raw.Now != nil {
		state.Now = raw.Now.track()
	}
	return state, nil
}
func (e *Engine) Spectrum() ([32]float64, error) {
	var bands [32]float64
	err := e.call("spectrum", nil, &bands)
	return bands, err
}
func (e *Engine) results(method string, params any) (SearchResults, error) {
	var raw struct {
		Songs, Albums, Playlists, Recent, Artists []wireTrack
		Err                                       string
	}
	if err := e.call(method, params, &raw); err != nil {
		return SearchResults{}, err
	}
	result := SearchResults{Songs: tracks(raw.Songs), Albums: tracks(raw.Albums), Playlists: tracks(raw.Playlists), Recent: tracks(raw.Recent), Artists: tracks(raw.Artists)}
	if raw.Err != "" {
		return result, errors.New(raw.Err)
	}
	return result, nil
}
func (e *Engine) Search(term string) (SearchResults, error) {
	return e.results("search", map[string]any{"query": term})
}
func (e *Engine) Library() (SearchResults, error) { return e.results("library", nil) }
func (e *Engine) Account() (Account, error) {
	var account Account
	err := e.call("account", nil, &account)
	return account, err
}
func (e *Engine) SignOut() error         { return e.call("sign_out", nil, nil) }
func (e *Engine) Reload() error          { return e.call("reload", nil, nil) }
func (e *Engine) PlayPause() error       { return e.call("play_pause", nil, nil) }
func (e *Engine) Next() error            { return e.call("next", nil, nil) }
func (e *Engine) Prev() error            { return e.call("prev", nil, nil) }
func (e *Engine) ToggleShuffle() error   { return e.call("toggle_shuffle", nil, nil) }
func (e *Engine) CycleRepeat() error     { return e.call("cycle_repeat", nil, nil) }
func (e *Engine) JumpTo(index int) error { return e.call("jump", map[string]any{"index": index}, nil) }
func (e *Engine) SeekTo(at time.Duration) error {
	return e.call("seek", map[string]any{"atMs": max(at.Milliseconds(), 0)}, nil)
}
func (e *Engine) SetVolume(volume int) error {
	return e.call("set_volume", map[string]any{"volume": min(max(volume, 0), 100)}, nil)
}
func (e *Engine) Play(kind, id string) error {
	return e.call("play", map[string]any{"kind": kind, "id": id}, nil)
}
func (e *Engine) QueueNext(kind, id string) error {
	return e.call("queue_next", map[string]any{"kind": kind, "id": id}, nil)
}
func (e *Engine) QueueLater(kind, id string) error {
	return e.call("queue_later", map[string]any{"kind": kind, "id": id}, nil)
}

func (e *Engine) EditQueue(action string, index, target int) error {
	return e.call("queue_edit", map[string]any{"action": action, "index": index, "target": target}, nil)
}
func (e *Engine) LibraryPage(category string, offset int) (SearchResults, error) {
	return e.results("library_page", map[string]any{"category": category, "offset": offset})
}
func (e *Engine) ArtistPage(id string) (SearchResults, error) {
	return e.results("artist_page", map[string]any{"id": id})
}

func (e *Engine) Discovery() (Discovery, error) {
	var feed Discovery
	err := e.call("discovery", nil, &feed)
	return feed, err
}

func (e *Engine) DiscoveryTracks(kind, id string) ([]Track, error) {
	var raw []wireTrack
	err := e.call("discovery_item", map[string]any{"kind": kind, "id": id}, &raw)
	return tracks(raw), err
}

func (e *Engine) SaveToLibrary(kind, id string) error {
	return e.call("save_item", map[string]any{"kind": kind, "id": id}, nil)
}

func (e *Engine) SavePlaylist(name string) error {
	_, err := e.results("save_playlist", map[string]any{"name": name})
	return err
}

func (e *Engine) Pause() error { return e.call("pause", nil, nil) }
