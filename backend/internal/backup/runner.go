package backup

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/periodic"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// DefaultInterval is the default backup interval (24 hours / daily).
const DefaultInterval = 24 * time.Hour

// Runner manages periodic database backup tasks.
type Runner struct {
	inner *periodic.Runner
}

// NewRunner creates a new backup runner with the given interval.
func NewRunner(db *sql.DB, objStore storage.ObjectStore, interval time.Duration) *Runner {
	task := func() {
		now := time.Now().UTC()
		key, err := Run(context.Background(), db, objStore, now)
		if err != nil {
			log.Printf("backup: run error: %v", err)
			return
		}
		log.Printf("backup: completed successfully, key: %s", key)
	}
	return &Runner{inner: periodic.NewRunner(interval, task)}
}

// Start begins the periodic backup loop in a goroutine.
// It runs a backup immediately on start, then on each tick.
func (r *Runner) Start() {
	r.inner.Start()
}

// Stop signals the runner to stop and waits for it to finish.
func (r *Runner) Stop() {
	r.inner.Stop()
}
