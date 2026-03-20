package cron

import (
	"database/sql"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/periodic"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// DefaultInterval is the default cron interval (1 hour).
const DefaultInterval = 1 * time.Hour

// Runner manages periodic cleanup tasks.
type Runner struct {
	inner *periodic.Runner
}

// NewRunner creates a new cron runner with the given interval.
func NewRunner(db *sql.DB, objStore storage.ObjectStore, interval time.Duration) *Runner {
	task := func() {
		if err := RunAll(db, objStore); err != nil {
			log.Printf("cron: run error: %v", err)
		}
	}
	return &Runner{inner: periodic.NewRunner(interval, task)}
}

// Start begins the periodic cleanup loop in a goroutine.
// It runs cleanup immediately on start, then on each tick.
func (r *Runner) Start() {
	r.inner.Start()
}

// Stop signals the runner to stop and waits for it to finish.
func (r *Runner) Stop() {
	r.inner.Stop()
}
