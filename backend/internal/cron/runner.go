package cron

import (
	"database/sql"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// DefaultInterval is the default cron interval (1 hour).
const DefaultInterval = 1 * time.Hour

// Runner manages periodic cleanup tasks.
type Runner struct {
	db       *sql.DB
	objStore storage.ObjectStore
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewRunner creates a new cron runner with the given interval.
func NewRunner(db *sql.DB, objStore storage.ObjectStore, interval time.Duration) *Runner {
	return &Runner{
		db:       db,
		objStore: objStore,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the periodic cleanup loop in a goroutine.
// It runs cleanup immediately on start, then on each tick.
func (r *Runner) Start() {
	go r.run()
}

func (r *Runner) run() {
	defer close(r.done)

	// Run immediately on start
	if err := RunAll(r.db, r.objStore); err != nil {
		log.Printf("cron: initial run error: %v", err)
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := RunAll(r.db, r.objStore); err != nil {
				log.Printf("cron: run error: %v", err)
			}
		case <-r.stop:
			return
		}
	}
}

// Stop signals the runner to stop and waits for it to finish.
func (r *Runner) Stop() {
	close(r.stop)
	<-r.done
}
