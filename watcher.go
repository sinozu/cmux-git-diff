package main

import (
	"log"
	"time"
)

// Watcher polls git diff at a regular interval and calls onChange
// when the diff content changes.
type Watcher struct {
	repoDir  string
	interval time.Duration
	lastHash string
	stop     chan struct{}
}

// NewWatcher creates a new Watcher for the given repository directory.
func NewWatcher(repoDir string, interval time.Duration) *Watcher {
	return &Watcher{
		repoDir:  repoDir,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Watch starts the polling loop. It calls onChange with the new DiffResult
// whenever the diff changes. This function blocks until Stop is called.
func (w *Watcher) Watch(onChange func(*DiffResult)) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial diff
	if result, err := GetDiff(w.repoDir); err == nil {
		w.lastHash = result.Hash
		onChange(result)
	}

	for {
		select {
		case <-ticker.C:
			result, err := GetDiff(w.repoDir)
			if err != nil {
				log.Printf("watcher: %v", err)
				continue
			}
			if result.Hash != w.lastHash {
				w.lastHash = result.Hash
				onChange(result)
			}
		case <-w.stop:
			return
		}
	}
}

// Stop signals the watcher to stop polling.
func (w *Watcher) Stop() {
	close(w.stop)
}
