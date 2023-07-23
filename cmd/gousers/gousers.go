// Simple application to read records from utmp/wtmp/btmp linux files
// File: "gousers.go"
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"gousers/exchange"
	"gousers/pkg/utmp"
	"gousers/pkg/vsnotify"
	"io"
	"log"
	"os"
	"time"
)

const DEBUG = false // log debug setting

const NOTIFY_INTERVAL = 100 * time.Millisecond // FIXME: why 100 ms?

// Options
var (
	Notify = false
	NoEUID = false
	File   = utmp.DefaultFile // "/var/run/utmp"
)

func Usage() {
	fmt.Print(`gousers - simple dump for utmp/wtmp/btmp linux files
Usage: gousers [options] [command]

Options:
  -help|--help - print full help
  -h|--h       - print help about options only
  -file <file> - use a specific file instead of /var/run/utmp
  -notify      - notify mode (Ctrl+C to stop)
  -noeuid      - don't use EUID (for wtmp/btmp)

Commands:
  user[s]         - show users is currently logged (default command)
  dump            - show full dump
  info <username> - show full information about user by username (JSON)
  stat            - show logged user statistics (JSON)

Example:
  gousers --help                           - print full help
  gousers [users]                          - show users from /var/run/utmp
  gousers dump                             - dump /var/run/utmp
  gousers info alice                       - show full information about user alice
  gousers stat                             - show logged user statistics
  gousers -file /var/log/btmp -noeuid user - show users from /var/run/btmp
  gousers -file /var/log/wtmp -noeuid dump - dump /var/log/wtmp
`)
	os.Exit(0)
} // func Usage()

func main() {
	// Check --help or -help options
	for _, opt := range os.Args[1:] {
		if opt == "-help" || opt == "--help" {
			Usage()
		} else if opt[0:1] != "-" {
			break // abort by first command
		}
	}

	// Parse options (flags)
	flag.StringVar(&File, "file", File, "Input utmp/wtmp/btmp file")
	flag.BoolVar(&Notify, "notify", Notify, "Notify mode (Ctrl+C to stop)")
	flag.BoolVar(&NoEUID, "noeuid", NoEUID, "don't use EUID (for wtmp/btmp)")
	flag.Parse()

	// Parse commands
	args := flag.Args() // os.Args without flags
	argc := len(args)

	// Define notify runner closure function
	runer := func(fn func(fname string, useEUID bool)) func(string, bool) {
		if !Notify {
			return fn
		}
		return func(fname string, useEUID bool) {
			fn(fname, useEUID)
			n, err := vsnotify.New(fname, NOTIFY_INTERVAL)
			if err != nil {
				log.Fatalf("fatal: %v", err)
			}
			for run := true; run; {
				select {
				case <-n.Update:
					fmt.Println()
					fn(fname, useEUID)
				case <-CtrlC:
					run = false
				}
			}
			n.Cancel()
		}
	}

	if argc == 0 { // show currently logged users by default
		runer(ShowUsers)(File, !NoEUID) // #1
		return
	}

	arg := args[0]

	if arg == "users" || arg == "user" { // show currently logged users
		runer(ShowUsers)(File, !NoEUID) // #2
	} else if arg == "info" { // show full information about user by username (JSON)
		if argc < 2 {
			log.Fatalf("fatal: no user selected (run with --help option)")
		} else {
			ShowUser(File, args[1], !NoEUID)
		}
	} else if arg == "stat" { // show logged user statistics (JSON)
		runer(ShowUsersStat)(File, !NoEUID)
	} else if arg == "dump" { // dump utmp/wtmp/btmp file
		DumpUtmp(File, Notify)
	} else { // show error and exit if command is unknown
		fmt.Fprintf(os.Stderr, "error: unknown command '%s' (run with --help option)\n", arg)
		os.Exit(1)
	}
} // func main()

// Show active users from utmp/wtmp/btmp file
func ShowUsers(fname string, useEUID bool) {
	users, err := utmp.Users(fname, useEUID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't read utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}

	for _, u := range users {
		utmp.UserPrint(os.Stdout, u)
	}
}

// Show Full user info
func ShowUser(fname, username string, useEUID bool) {
	users, err := utmp.Users(fname, useEUID)
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
	u := exchange.User{
		Name:        uf.Name,
		UID:         uf.UID,
		GID:         uf.GID,
		DisplayName: uf.DisplayName,
		HomeDir:     uf.HomeDir,
		Groups:      uf.Groups,
		LogonType:   exchange.LogonType[uf.Type],
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
func ShowUsersStat(fname string, useEUID bool) {
	users, err := utmp.Users(fname, useEUID)
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
