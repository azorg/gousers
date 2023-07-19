// Simple application to read records from utmp/wtmp/btmp linux files
// File: "gousers.go"
package main

import (
	"errors"
	"flag"
	"fmt"
	"gousers/pkg/utmp"
	"io"
	"os"
)

func Usage() {
	fmt.Print(`gousers - simple dump for utmp/wtmp/btmp linux files
Usage: gousers [options] [command]

Options:
  -h|--h    - print help about options
  -f <file> - use a specific file instead of /var/run/utmp

Commands:
  user[s] - show users is currently logged (default command)
  dump    - show full dump

Example:
  gousers --help|-help          - print help
  gousers [users]               - show users from /var/run/utmp
  gousers -f /var/log/btmp user - show users from /var/run/btmp
  gousers dump                  - dump /var/run/utmp
  gousers -f /var/log/wtmp dump - dump /var/log/wtmp
`)
	os.Exit(0)
} // func Usage()

func BadCommand() {
	fmt.Fprintln(os.Stderr, "error: bad command")
	fmt.Fprintln(os.Stderr, "Try gousers --help' for more information.")
	os.Exit(1)
} // func badCommand()

func main() {
	// Options
	fname := utmp.DefaultFile // "/var/run/utmp"

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
	flag.Parse()

	// Parse commands
	args := flag.Args() // os.Args without flags
	argc := len(args)

	if argc == 0 {
		ShowUsers(fname) // show currently logged users by default
		return
	}

	arg := args[0]

	if arg == "users" || arg == "user" {
		ShowUsers(fname) // show currently logged users
	} else if arg == "dump" {
		DumpUtmp(fname) // dump utmp/wtmp/btmp file
	} else {
		BadCommand() // show error and exit if command is unknown
	}
} // func main()

// Show active users from utmp.wtmp file
func ShowUsers(fname string) {
	users, err := utmp.Users(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't read utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}

	for _, u := range users {
		utmp.UserPrint(os.Stdout, u)
	}
} // func ShowUsers()

// Dump utmp/wtmp/btmp file as plain text
func DumpUtmp(fname string) {
	f, err := os.Open(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't open utmp/wtmp/btmp file: %v\n", err)
		os.Exit(2)
	}
	defer f.Close()

	for {
		var u utmp.Utmp
		err = utmp.Read(f, &u)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			} else {
				fmt.Fprintln(os.Stderr, `error: read "%s": %v`, fname, err)
				os.Exit(3)
			}
		}
		utmp.Print(os.Stdout, u)
	} // for
} // func DumpUtmp

// EOF: "gousers.go"
