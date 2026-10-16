//go:build windows

package mpris

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var registerHotKey = user32.NewProc("RegisterHotKey")
var unregisterHotKey = user32.NewProc("UnregisterHotKey")
var getMessage = user32.NewProc("GetMessageW")
var peekMessage = user32.NewProc("PeekMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")
var getThreadID = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")

type windowsMessage struct {
	Window  uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	X, Y    int32
	Private uint32
}
type Server struct {
	thread uintptr
	once   sync.Once
	done   chan struct{}
}

func Publish(ctl Controls) (*Server, error) {
	s := &Server{done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(s.done)
		s.thread, _, _ = getThreadID.Call()
		var msg windowsMessage
		peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
		count := 0
		for i, key := range []uintptr{0xB3, 0xB0, 0xB1} {
			ok, _, _ := registerHotKey.Call(0, uintptr(i+1), 0x4000, key)
			if ok == 0 {
				for j := 1; j <= count; j++ {
					unregisterHotKey.Call(0, uintptr(j))
				}
				ready <- fmt.Errorf("media keys are already owned by another app")
				return
			}
			count++
		}
		defer func() {
			for i := 1; i <= 3; i++ {
				unregisterHotKey.Call(0, uintptr(i))
			}
		}()
		ready <- nil
		for {
			result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(result) <= 0 {
				return
			}
			if msg.Message == 0x0312 {
				switch msg.WParam {
				case 1:
					_ = ctl.PlayPause()
				case 2:
					_ = ctl.Next()
				case 3:
					_ = ctl.Prev()
				}
			}
		}
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Server) Update(State) {}
func (s *Server) Close() {
	if s != nil {
		s.once.Do(func() { postThreadMessage.Call(s.thread, 0x12, 0, 0) })
	}
}
