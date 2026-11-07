package engine

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// This opt-in test uses an isolated copy of the account, mutes playback, and
// cuts one encrypted CDN connection. The proxy tunnels TLS without decrypting it.
func TestNativeLiveRecoversInterruptedDownload(t *testing.T) {
	if os.Getenv("NATIVE_RECOVERY_LIVE_TEST") != "1" {
		t.Skip("requires explicit opt-in and an existing signed-in subscription")
	}
	profile := os.Getenv("POMME_DATA_DIR")
	if profile == "" {
		profile = filepath.Join(os.Getenv("LOCALAPPDATA"), "PommeTUI")
	}
	credentials, err := os.ReadFile(filepath.Join(profile, "cache", "apple", "credentials.json"))
	if err != nil {
		t.Fatal("the playback test requires an existing signed-in Pomme profile")
	}
	isolated := t.TempDir()
	account := filepath.Join(isolated, "cache", "apple")
	if err := os.MkdirAll(account, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(account, "credentials.json"), credentials, 0600); err != nil {
		t.Fatal("could not prepare the isolated account")
	}
	t.Setenv("POMME_DATA_DIR", isolated)
	t.Setenv("SONORA_APPLE_MEDIA_USER_TOKEN", "")
	t.Setenv("SONORA_APPLE_BEARER_TOKEN", "")
	proxy, dropped := interruptingProxy(t)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		t.Setenv(key, proxy)
	}
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")
	path, err := backendPath()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := start(path)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	var hello struct{ Authed bool }
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := eng.rpc.call(ctx, "hello", nil, &hello); err != nil || !hello.Authed {
		t.Fatal("the isolated session could not authenticate")
	}
	if err := eng.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	if err := eng.Play("song", "1535067323"); err != nil {
		t.Fatal(err)
	}
	var started, recovering, recovered bool
	var peak time.Duration
	deadline := time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		state, err := eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if state.Err != "" {
			t.Fatal(state.Err)
		}
		if state.Now.ID != "1535067323" || state.QueuePos != 0 || state.Volume != 0 {
			t.Fatal("recovery changed the track, queue position, or mute setting")
		}
		if state.Playing && !recovering {
			started = true
			peak = max(peak, state.Pos)
		}
		if state.Recovering && started {
			recovering = true
		}
		if recovering && state.Playing && !state.Recovering && state.Pos >= peak-time.Second {
			recovered = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !dropped.Load() || !started || !recovering || !recovered || peak < time.Second {
		t.Fatalf("interruption was not recovered: dropped=%t started=%t recovering=%t recovered=%t last_position=%s", dropped.Load(), started, recovering, recovered, peak)
	}
	t.Logf("recovered the same muted song automatically at approximately %s", peak.Round(time.Second))
	// Confirm that pausing and reopening still preserves the position after recovery.
	if err := eng.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := eng.SeekTo(12 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := eng.Reload(); err != nil {
		t.Fatal(err)
	}
	state, err := eng.State()
	if err != nil || state.Playing || state.Initializing || state.Pos != 12*time.Second {
		t.Fatal("paused reload failed after automatic recovery")
	}
	// Exercise an ordinary near-end transition too, including duplicate catalog IDs.
	if err := eng.QueueLater("song", "1535067323"); err != nil {
		t.Fatal(err)
	}
	if state.Dur <= time.Second {
		t.Fatal("the live track has no usable duration")
	}
	if err := eng.SeekTo(state.Dur - 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := eng.PlayPause(); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		state, err = eng.State()
		if err != nil || state.Err != "" {
			t.Fatal("playback failed at the near-end queue transition")
		}
		if state.QueuePos == 1 && state.Playing && state.Volume == 0 {
			t.Log("near-end transition started the next queued song while remaining muted")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("the next queued song did not start after seeking near the end")
}

// Throttle the first media download so it reaches audible playback, then cut it.
// Every later connection passes through normally, allowing a real automatic retry.
func interruptingProxy(t *testing.T) (string, *atomic.Bool) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	dropped := &atomic.Bool{}
	var mu sync.Mutex
	connections := make(map[net.Conn]struct{})
	server := &http.Server{ReadHeaderTimeout: 10 * time.Second}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "CONNECT required", http.StatusMethodNotAllowed)
			return
		}
		upstream, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
		if err != nil {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		client, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		mu.Lock()
		connections[client], connections[upstream] = struct{}{}, struct{}{}
		mu.Unlock()
		defer func() {
			mu.Lock()
			delete(connections, client)
			delete(connections, upstream)
			mu.Unlock()
		}()
		_, _ = buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		if buffered.Flush() != nil {
			return
		}
		go func() { _, _ = io.Copy(upstream, buffered); _ = upstream.Close() }()
		media := strings.Contains(r.Host, "audio-") || strings.Contains(r.Host, "aod-")
		if !media || dropped.Load() {
			_, _ = io.Copy(client, upstream)
			return
		}
		buffer := make([]byte, 16*1024)
		var transferred int
		for {
			n, readErr := upstream.Read(buffer)
			if n > 0 {
				if _, err := client.Write(buffer[:n]); err != nil {
					return
				}
				transferred += n
				if transferred >= 512*1024 && dropped.CompareAndSwap(false, true) {
					return
				}
				if !dropped.Load() {
					time.Sleep(250 * time.Millisecond)
				}
			}
			if readErr != nil {
				return
			}
		}
	})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		mu.Lock()
		defer mu.Unlock()
		for connection := range connections {
			_ = connection.Close()
		}
	})
	return "http://" + listener.Addr().String(), dropped
}
