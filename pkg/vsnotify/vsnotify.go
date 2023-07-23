/*
Package vsnotify implement Very Simply file change notifier.

File: "vsnotify.go"
*/
package vsnotify

import (
	"log"
	"os"
	"time"
)

const DEBUG = true

type Watcher interface {
	Evt() <-chan time.Time // return modification file time
	Close()                // terminate watcher
}

type Watch struct {
	updt chan time.Time
	done chan bool
}

func NewWatcher(filePath string, period time.Duration) (Watcher, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	var w Watch
	w.updt = make(chan time.Time)
	w.done = make(chan bool)

	go func() {
		var pstat os.FileInfo
		tick := time.Tick(period)
		for stop := false; !stop; pstat = stat {
			stat, err = os.Stat(filePath)
			if err != nil {
				log.Printf("error: %v", err)
				break
			}

			if pstat != nil {
				if stat.Size() != pstat.Size() || stat.ModTime() != pstat.ModTime() {
					w.updt <- stat.ModTime() // file updated
				}
			}

			select {
			case <-tick:
			case <-w.done:
				stop = true
			}
		}
		if DEBUG {
			log.Printf("Watcher(%s) done", filePath)
		}
		close(w.updt)
	}()

	return &w, nil
}

// Get chanel to select update file time
func (w *Watch) Evt() <-chan time.Time {
	return w.updt
}

// Close file watcher
func (w *Watch) Close() {
	if w != nil {
		close(w.done)
	}
}

// EOF: "vsnotify.go"
