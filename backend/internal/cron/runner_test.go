package cron_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/cron"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestRunner_StartStop(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	memStore := storage.NewMemoryStore()

	runner := cron.NewRunner(db, memStore, 50*time.Millisecond)
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
}

func TestNewRunner_DefaultInterval(t *testing.T) {
	t.Parallel()
	if cron.DefaultInterval != 1*time.Hour {
		t.Errorf("expected DefaultInterval=1h, got %v", cron.DefaultInterval)
	}
}
