package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Child-mode fixture acts as an unauthenticated backend and a long-lived login window owner.
func init() {
	if os.Getenv("AMTUI_FAKE_LOGIN_CHILD") != "1" {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "login" {
		until := time.Now().Add(10 * time.Second)
		for time.Now().Before(until) {
			_ = os.WriteFile(os.Getenv("AMTUI_FAKE_LOGIN_MARKER"), []byte("alive"), 0600)
			time.Sleep(30 * time.Millisecond)
		}
		os.Exit(0)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(scanner.Bytes(), &request)
		result := any(true)
		if request.Method == "hello" {
			result = map[string]any{"protocol": 1, "authed": false}
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"id": request.ID, "result": result})
		if request.Method == "shutdown" {
			os.Exit(0)
		}
	}
	os.Exit(0)
}
func TestConnectCancellationTerminatesOwnedLoginHelper(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "login-alive")
	t.Setenv("AMTUI_NATIVE_BACKEND", executable)
	t.Setenv("AMTUI_FAKE_LOGIN_CHILD", "1")
	t.Setenv("AMTUI_FAKE_LOGIN_MARKER", marker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := time.Now()
	canceledAt := make(chan time.Time, 1)
	go func() {
		// Windows may scan a newly built executable before letting it start.
		// Measure cancellation after the helper starts, separately from startup.
		for time.Since(started) < 15*time.Second {
			if _, err := os.Stat(marker); err == nil {
				canceledAt <- time.Now()
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		canceledAt <- time.Now()
		cancel()
	}()
	eng, err := ConnectContext(ctx, func(string) {})
	if eng != nil {
		eng.Close()
	}
	if err == nil {
		t.Fatal("canceled connection succeeded")
	}
	if time.Since(<-canceledAt) > 2*time.Second {
		t.Fatal("cancellation left login helper running")
	}
	first, err := os.Stat(marker)
	if err != nil {
		t.Fatal("login helper never started", err)
	}
	time.Sleep(150 * time.Millisecond)
	last, err := os.Stat(marker)
	if err != nil || !first.ModTime().Equal(last.ModTime()) {
		t.Fatal("login helper survived cancellation")
	}
}
