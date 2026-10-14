package main

import (
	"testing"
	"time"
)

func TestConnectionLifecycleCancelsUnreadyTasksOnQuit(t *testing.T) {
	lifecycle := newConnectionLifecycle()
	if !lifecycle.begin() {
		t.Fatal("initial startup rejected")
	}
	done := make(chan struct{})
	go func() { <-lifecycle.ctx.Done(); lifecycle.finish(nil); close(done) }()
	lifecycle.shutdown()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("startup task survived quit")
	}
	if lifecycle.begin() {
		t.Fatal("a new connection started after shutdown")
	}
	lifecycle.shutdown()
}
