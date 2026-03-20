package periodic

import "time"

// Runner executes a task function periodically at a fixed interval.
// It runs the task immediately on Start, then repeats on each tick.
type Runner struct {
	task     func()
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewRunner creates a new periodic runner with the given interval and task.
func NewRunner(interval time.Duration, task func()) *Runner {
	return &Runner{
		task:     task,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the periodic loop in a goroutine.
func (r *Runner) Start() {
	go r.run()
}

func (r *Runner) run() {
	defer close(r.done)

	// Run immediately on start
	r.task()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.task()
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
