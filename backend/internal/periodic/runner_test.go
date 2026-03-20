package periodic_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/periodic"
)

func TestRunner_StartStop(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	runner := periodic.NewRunner(50*time.Millisecond, func() {
		count.Add(1)
	})
	runner.Start()

	// Let it run a couple ticks
	time.Sleep(120 * time.Millisecond)

	// Stop should return without hanging
	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Runner.Stop() did not return within 2 seconds")
	}

	// Should have run at least twice (immediate + at least one tick)
	if c := count.Load(); c < 2 {
		t.Errorf("expected at least 2 runs, got %d", c)
	}
}

func TestRunner_ImmediateRun(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	runner := periodic.NewRunner(10*time.Second, func() {
		count.Add(1)
	})
	runner.Start()

	// Wait a short time — should have run once immediately
	time.Sleep(50 * time.Millisecond)
	runner.Stop()

	if c := count.Load(); c != 1 {
		t.Errorf("expected exactly 1 immediate run, got %d", c)
	}
}
