// Package vsnotify implement Very Simply file change notifier.
// File: "vsnotify.go"
package vsnotify

import (
	"log"
	"os"
	"sync"
	"time"
)

var DEBUG = true // debug output to log on terminate

type Notify struct {
	Update <-chan time.Time // return modification file time
	Cancel func()           // terminate notifier
}

// Create new file change notifier
func New(filePath string, period time.Duration) (*Notify, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	update := make(chan time.Time)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

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
					update <- stat.ModTime() // file updated
				}
			}

			select {
			case <-tick:
			case <-done:
				stop = true
			}
		}
		close(update)
		wg.Done()
	}()

	cancel := func() {
		close(done)
		wg.Wait()
		if DEBUG {
			log.Printf("vsnotifier(%s) done", filePath)
		}
	}

	return &Notify{update, cancel}, nil
}

// EOF: "vsnotify.go"
