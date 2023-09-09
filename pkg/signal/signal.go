// Packege signal implement some signal bridge to channels.
// File: "signal.go"
package signal

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	CtrlC  chan struct{} // Ctrl+C -> SIGINT or SIGTERM
	CtrlZ  chan struct{} // Ctrl+Z -> SIGTSTP
	CtrlBS chan struct{} // Ctrl+\ -> SIGQUIT
)

// Setup Ctrl+C | Ctrl+Z | Ctrl+\ channels
func init() {
	CtrlC = make(chan struct{}, 1)
	CtrlZ = make(chan struct{}, 1)
	CtrlBS = make(chan struct{}, 1)
	ch := make(chan os.Signal, 1)

	sigList := []os.Signal{
		//syscall.SIGTERM,
		syscall.SIGINT,  // Ctrl-C
		syscall.SIGTSTP, // Ctrl-Z
		syscall.SIGQUIT, // Ctrl-\
		//syscall.SIGHUP,
	}

	//signal.Ignore(sigList...)
	signal.Notify(ch, sigList...)

	go func() {
		for sig := range ch {
			fmt.Fprint(os.Stderr, "\r\n")
			switch sig {
			case syscall.SIGTERM:
				log.Print(`SIGTERM received`)
			case syscall.SIGINT:
				log.Print(`SIGINT received (Ctrl+C pressed)`)
				if len(CtrlC) == 0 {
					CtrlC <- struct{}{}
				}
			case syscall.SIGTSTP:
				log.Print(`SIGTSTP received (Ctrl+Z pressed)`)
				if len(CtrlZ) == 0 {
					CtrlZ <- struct{}{}
				}
			case syscall.SIGQUIT:
				log.Print(`SIGQUIT received (Ctrl+\ pressed)`)
				if len(CtrlBS) == 0 {
					CtrlBS <- struct{}{}
				}
      case syscall.SIGHUP:
				log.Print(`SIGHUP received`)
				//...
			default:
				log.Printf("unknown signal=%v received", sig)
			} // switch
		} // for
	}()
}

// Debug wait
func WaitCtrl() {
	fmt.Println(`press Ctrl+\ to resume or Ctrl+C to abort`)
	select {
	case <-CtrlBS:
		log.Print(`resume application by Ctrl+\`)
	case <-CtrlC:
		log.Fatal("abort application by Ctrl+С")
	}
} // func WaitCtr()

// EOF: "signal.go"
