package main

import (
	"context"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"sync"
)

// connectionLifecycle owns startup/reconnect tasks even before a ready engine reaches the model.
type connectionLifecycle struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	closed  bool
	tasks   sync.WaitGroup
	engines map[*engine.Engine]struct{}
}

func newConnectionLifecycle() *connectionLifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &connectionLifecycle{ctx: ctx, cancel: cancel, engines: make(map[*engine.Engine]struct{})}
}
func (l *connectionLifecycle) begin() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.tasks.Add(1)
	return true
}
func (l *connectionLifecycle) finish(eng *engine.Engine) {
	l.mu.Lock()
	if eng != nil {
		l.engines[eng] = struct{}{}
	}
	closed := l.closed
	l.mu.Unlock()
	if closed && eng != nil {
		eng.Close()
	}
	l.tasks.Done()
}
func (l *connectionLifecycle) shutdown() {
	l.mu.Lock()
	l.closed = true
	l.cancel()
	l.mu.Unlock()
	l.tasks.Wait()
	l.mu.Lock()
	engines := make([]*engine.Engine, 0, len(l.engines))
	for eng := range l.engines {
		engines = append(engines, eng)
	}
	l.mu.Unlock()
	for _, eng := range engines {
		eng.Close()
	}
}
