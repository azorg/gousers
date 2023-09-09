// File: "log.go"

package main

import (
	"log"
	"os"
)

func init() {
	// Setup standart logger
	log.SetOutput(os.Stderr)
	log.SetPrefix("")
	if DEBUG {
		log.SetFlags(log.Lshortfile)
	} else {
		log.SetFlags(0)
	}
}

// EOF: "log.go"
