package backup

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// DefaultInterval is the default backup interval (24 hours / daily).
const DefaultInterval = 24 * time.Hour

// Runner manages periodic database backup tasks.
type Runner struct {
	db       *sql.DB
	objStore storage.ObjectStore
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewRunner creates a new backup runner with the given interval.
func NewRunner(db *sql.DB, objStore storage.ObjectStore, interval time.Duration) *Runner {
	return &Runner{
		db:       db,
		objStore: objStore,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the periodic backup loop in a goroutine.
// It runs a backup immediately on start, then on each tick.
func (r *Runner) Start() {
	go r.run()
}

func (r *Runner) run() {
	defer close(r.done)

	// Run immediately on start
	r.runBackup()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.runBackup()
		case <-r.stop:
			return
		}
	}
}

func (r *Runner) runBackup() {
	now := time.Now().UTC()
	key, err := Run(context.Background(), r.db, r.objStore, now)
	if err != nil {
		log.Printf("backup: run error: %v", err)
		return
	}
	log.Printf("backup: completed successfully, key: %s", key)
}

// Stop signals the runner to stop and waits for it to finish.
func (r *Runner) Stop() {
	close(r.stop)
	<-r.done
}
