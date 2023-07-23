// Simple application to read records from utmp/wtmp/btmp linux files
// File: "gousers.go"
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	EX "gousers/exchange"
	"gousers/pkg/utmp"
	"gousers/pkg/vsnotify"
	"io"
	"log"
	"os"
	"time"
)

const DEBUG = false
const NOTIFY_INTERVAL = 100 * time.Millisecond // FIXME: why 100 ms?

func Usage() {
	fmt.Print(`gousers - simple dump for utmp/wtmp/btmp linux files
Usage: gousers [options] [command]

Options:
  -help|--help - print full help
  -h|--h       - print help about options only
  -f <file>    - use a specific file instead of /var/run/utmp
  -n           - notify mode (Ctrl+C to stop)

Commands:
  user[s]         - show users is currently logged (default command)
  dump            - show full dump
  info <username> - show full information about user by username (JSON)
  stat            - show logged user statistics (JSON)

Example:
  gousers --help                - print full help
  gousers [users]               - show users from /var/run/utmp
  gousers dump                  - dump /var/run/utmp
  gousers info alice            - show full information about user alice
  gousers stat                  - show logged user statistics
  gousers -f /var/log/btmp user - show users from /var/run/btmp
  gousers -f /var/log/wtmp dump - dump /var/log/wtmp
`)
	os.Exit(0)
} // func Usage()

func main() {
	// Options
	fname := utmp.DefaultFile // "/var/run/utmp"
	notify := false

	// Check --help or -help options
	for _, opt := range os.Args[1:] {
		if opt == "-help" || opt == "--help" {
			Usage()
		} else if opt[0:1] != "-" {
			break // abort by first command
		}
	}

	// Parse options (flags)
	flag.StringVar(&fname, "f", fname, "Input utmp/wtmp/btmp file")
	flag.BoolVar(&notify, "n", notify, "Notify mode (users/stat)")
	flag.Parse()

	// Parse commands
	args := flag.Args() // os.Args without flags
	argc := len(args)

	// Define notify runner closure function
	runer := func(fn func(fname string)) func(string) {
		if !notify {
			return fn
		}
		return func(fname string) {
			fn(fname)
			w, err := vsnotify.NewWatcher(fname, NOTIFY_INTERVAL)
			if err != nil {
				log.Fatalf("fatal: %v", err)
			}
			for run := true; run; {
				select {
				case <-w.Evt():
					fmt.Println()
					fn(fname)
				case <-CtrlC:
					run = false
				}
			}
			w.Close()
		}
	}

	if argc == 0 { // show currently logged users by default
		runer(ShowUsers)(fname) // #1
		return
	}

	arg := args[0]

	if arg == "users" || arg == "user" { // show currently logged users
		runer(ShowUsers)(fname) // #2
	} else if arg == "info" { // show full information about user by username (JSON)
		if argc < 2 {
			log.Fatalf("fatal: no user selected (run with --help option)")
		} else {
			ShowUser(fname, args[1])
		}
	} else if arg == "stat" { // show logged user statistics (JSON)
		runer(ShowUsersStat)(fname)
	} else if arg == "dump" { // dump utmp/wtmp/btmp file
		DumpUtmp(fname, notify)
	} else { // show error and exit if command is unknown
		fmt.Fprintf(os.Stderr, "error: unknown command '%s' (run with --help option)\n", arg)
		os.Exit(1)
	}
} // func main()

// Show active users from utmp/wtmp/btmp file
func ShowUsers(fname string) {
	users, err := utmp.Users(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't read utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}

	for _, u := range users {
		utmp.UserPrint(os.Stdout, u)
	}
}

// Show Full user info
func ShowUser(fname, username string) {
	users, err := utmp.Users(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't read utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}

	uf, err := utmp.GetUserFull(users, username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	// Repack utmp.UserFull to exchange.User
	u := EX.User{
		Name:        uf.Name,
		UID:         uf.UID,
		GID:         uf.GID,
		DisplayName: uf.DisplayName,
		HomeDir:     uf.HomeDir,
		Groups:      uf.Groups,
		LogonType:   EX.LogonType[uf.Type],
		LogonTime:   uf.Time,
		Logons:      uf.Logons}

	// Encode full user info to JSON
	data, err := json.MarshalIndent(&u, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: json.Marshal():", err)
		return
	}

	fmt.Println(string(data))
}

// Show logged user statistics (JSON)
func ShowUsersStat(fname string) {
	users, err := utmp.Users(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't read utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}

	// get logged user statistics
	us := utmp.GetUsersStat(users)

	// Encode statistics to JSON
	data, err := json.MarshalIndent(&us, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: json.Marshal():", err)
		return
	}

	fmt.Println(string(data))
} // func ShowUsersStat()

// Dump utmp/wtmp/btmp file as plain text
func DumpUtmp(fname string, notify bool) {
	f, err := os.Open(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't open utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}
	defer f.Close()

	for stop := false; !stop; {
		var u utmp.Utmp
		err = utmp.Read(f, &u)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				log.Fatalf(`fatal: read "%s": %v`, fname, err)
			}

			if !notify {
				break
			}

			select {
			case <-time.After(NOTIFY_INTERVAL):
			case <-CtrlC:
				stop = true
			}
			continue
		}

		utmp.Print(os.Stdout, u)
	} // for
} // func DumpUtmp

// EOF: "gousers.go"
