package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type rpcReply struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}
type rpcClient struct {
	reader   io.ReadCloser
	writer   io.WriteCloser
	serial   atomic.Uint64
	mu       sync.Mutex
	pending  map[uint64]chan rpcReply
	outgoing chan []byte
	done     chan struct{}
	once     sync.Once
}

func newRPC(reader io.ReadCloser, writer io.WriteCloser) *rpcClient {
	r := &rpcClient{reader: reader, writer: writer, pending: make(map[uint64]chan rpcReply), outgoing: make(chan []byte, 64), done: make(chan struct{})}
	go r.readLoop()
	go r.writeLoop()
	return r
}
func (r *rpcClient) readLoop() {
	scanner := bufio.NewScanner(r.reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var response rpcReply
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			r.close()
			return
		}
		r.mu.Lock()
		ch := r.pending[response.ID]
		r.mu.Unlock()
		if ch != nil {
			select {
			case ch <- response:
			default:
			}
		}
	}
	r.close()
}
func (r *rpcClient) writeLoop() {
	for {
		select {
		case <-r.done:
			return
		case data := <-r.outgoing:
			if _, err := r.writer.Write(data); err != nil {
				r.close()
				return
			}
		}
	}
}

// call correlates independent requests; a slow catalog lookup cannot hold a transport lock.
func (r *rpcClient) call(ctx context.Context, method string, params any, out any) error {
	if params == nil {
		params = map[string]any{}
	}
	id := r.serial.Add(1)
	data, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("request exceeds size limit")
	}
	data = append(data, '\n')
	ch := make(chan rpcReply, 1)
	r.mu.Lock()
	r.pending[id] = ch
	r.mu.Unlock()
	defer func() { r.mu.Lock(); delete(r.pending, id); r.mu.Unlock() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return errors.New("native backend closed")
	case r.outgoing <- data:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return errors.New("native backend closed")
	case response := <-ch:
		if response.Error != nil {
			return fmt.Errorf("native backend: %s", *response.Error)
		}
		if out != nil {
			return json.Unmarshal(response.Result, out)
		}
		return nil
	}
}
func (r *rpcClient) close() {
	r.once.Do(func() { close(r.done); _ = r.reader.Close(); _ = r.writer.Close() })
}
