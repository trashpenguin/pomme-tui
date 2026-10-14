package visualizer

import (
	"errors"
	"testing"
	"time"
)

func TestNativeSpectrumPublishesLiveFrameAndCloses(t *testing.T) {
	bands := [32]float64{0.25, 0.5, 1}
	service, err := NewNativeService(func() ([32]float64, error) { return bands, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if frame, ok := service.Latest(); ok {
			if frame.Bands != bands || !frame.Live || frame.Source != "native playback" || frame.At.IsZero() {
				t.Fatalf("bad native frame: %+v", frame)
			}
			if err := service.Close(); err != nil {
				t.Fatal(err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("native spectrum never published")
}
func TestNativeSpectrumReportsBackendFailure(t *testing.T) {
	service, err := NewNativeService(func() ([32]float64, error) { return [32]float64{}, errors.New("backend closed") })
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if service.Err() != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("backend failure swallowed")
}
