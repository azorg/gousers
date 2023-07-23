/*
Package vsnotify implement Very Simply file change notifier.

File: "vsnotify.go"
*/
package vsnotify

import (
	"log"
	"os"
	"time"
  "sync"
)

var DEBUG = true // debug output to log on terminate

type Notify struct {
	Update chan time.Time // return modification file time
	Cancel func()         // terminate notifier
}

// Create new file change notifier
func New(filePath string, period time.Duration) (*Notify, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

  var n Notify
  n.Update = make(chan time.Time)
  cancel := make(chan struct{})
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
					n.Update <- stat.ModTime() // file updated
				}
			}

			select {
			case <-tick:
			case <-cancel:
				stop = true
			}
		}
		close(n.Update)
    wg.Done()
	}()

  n.Cancel = func() {
		close(cancel)
    wg.Wait()
		if DEBUG {
			log.Printf("vsnotifier(%s) done", filePath)
		}
  }

	return &n, nil
}

// EOF: "vsnotify.go"
