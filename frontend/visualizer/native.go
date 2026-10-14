package visualizer

import (
	"context"
	"errors"
	"math"
	"time"
)

// NewNativeService publishes already analyzed bands, without a system capture source or extra FFT.
func NewNativeService(read func() ([32]float64, error)) (*Service, error) {
	if read == nil {
		return nil, errors.New("native spectrum reader is nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{ctx: ctx, cancel: cancel, done: make(chan struct{}), sourceName: "native playback"}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(time.Second / 30)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			bands, err := read()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				s.setTerminalError(err)
				return
			}
			for i, value := range bands {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					bands[i] = 0
				} else {
					bands[i] = min(max(value, 0), 1)
				}
			}
			s.latest.Store(&Frame{Bands: bands, Live: true, Source: s.sourceName, At: time.Now()})
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return s, nil
}
