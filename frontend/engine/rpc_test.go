package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

func TestRPCMatchesOutOfOrderResponses(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	rpc := newRPC(client, client)
	defer rpc.close()
	go func() {
		scanner := bufio.NewScanner(server)
		requests := []map[string]any{}
		for len(requests) < 2 && scanner.Scan() {
			var req map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &req)
			requests = append(requests, req)
		}
		for i := len(requests) - 1; i >= 0; i-- {
			_ = json.NewEncoder(server).Encode(map[string]any{"id": requests[i]["id"], "result": requests[i]["method"]})
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, method := range []string{"search", "state"} {
		wg.Add(1)
		go func(method string) {
			defer wg.Done()
			var got string
			if err := rpc.call(ctx, method, map[string]any{}, &got); err != nil {
				t.Error(err)
				return
			}
			if got != method {
				t.Errorf("reply %q for %q", got, method)
			}
		}(method)
	}
	wg.Wait()
}
func TestRPCSlowSearchDoesNotBlockState(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	rpc := newRPC(client, client)
	defer rpc.close()
	go func() {
		scanner := bufio.NewScanner(server)
		for scanner.Scan() {
			var req map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &req)
			if req["method"] == "state" {
				_ = json.NewEncoder(server).Encode(map[string]any{"id": req["id"], "result": true})
			}
		}
	}()
	searchCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = rpc.call(searchCtx, "search", map[string]any{}, nil); close(done) }()
	ctx, stop := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer stop()
	var result bool
	if err := rpc.call(ctx, "state", map[string]any{}, &result); err != nil || !result {
		t.Fatalf("state blocked: %v", err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation leaked request")
	}
}
func TestRPCBackendDeathFailsPendingCallAndCloseIsIdempotent(t *testing.T) {
	client, server := net.Pipe()
	rpc := newRPC(client, client)
	go func() { scanner := bufio.NewScanner(server); scanner.Scan(); server.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rpc.call(ctx, "state", map[string]any{}, nil); err == nil {
		t.Fatal("EOF accepted")
	}
	rpc.close()
	rpc.close()
}
