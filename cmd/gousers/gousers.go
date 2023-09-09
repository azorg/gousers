// Simple application to read records from utmp/wtmp/btmp linux files
// File: "gousers.go"
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gousers/dto"
	"gousers/pkg/signal"
	"gousers/pkg/utmp"
)

const DEBUG = true

const FOLLOW_INTERVAL = 250 * time.Millisecond // FIXME: why 250 ms?

// Options (default values)
var (
	Follow  = false
	UseEUID = false
	File    = "/var/log/wtmp"
)

func Usage() {
	fmt.Print(`gousers - simple dump for utmp/wtmp/btmp linux files
Usage: gousers [options] [command]

Options:
  -help|--help - print full help
  -h|--h       - print help about options only
  -file <file> - use a specific file instead of /var/log/wtmp
  -follow      - follow dump mode (Ctrl+C to stop) like "tail -f"
  -euid        - use EUID (for utmp)

Commands:
  user[s]         - show users is currently logged (default command)
  dump            - show full dump
  info <username> - show full information about user by username (JSON)
  stat            - show logged user statistics (JSON)
  monitor         - login/logout monitor

Example:
  gousers --help                           - print full help
  gousers [users]                          - show users from /var/run/utmp
  gousers dump                             - dump /var/run/utmp
  gousers info alice                       - show full information about user alice
  gousers stat                             - show logged user statistics
  gousers -file /var/log/btmp -noeuid user - show users from /var/log/btmp
  gousers -file /var/log/wtmp -noeuid dump - dump /var/log/wtmp
  gousers -file /var/run/utmp              - show users from /var/run/utmp
  gousers -follow dump                     - follow dump /var/log/wtmp
`)
	os.Exit(0)
}

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
	flag.BoolVar(&Follow, "follow", Follow, "Follow dump mode (Ctrl+C to stop)")
	flag.BoolVar(&UseEUID, "euid", UseEUID, "use EUID (for utmp)")
	flag.Parse()

	// Parse commands
	args := flag.Args() // os.Args without flags
	argc := len(args)

	if argc == 0 { // show currently logged users by default
		ShowUsers(File, UseEUID) // #1
		return
	}

	arg := args[0]

	if arg == "users" || arg == "user" { // show currently logged users
		ShowUsers(File, UseEUID) // #2
	} else if arg == "info" { // show full information about user (JSON)
		if argc < 2 {
			log.Fatalf("fatal: no user selected (run with --help option)")
		} else {
			ShowUser(File, args[1], UseEUID)
		}
	} else if arg == "stat" { // show logged user statistics (JSON)
		ShowUsersStat(File, UseEUID)
	} else if arg == "dump" { // dump utmp/wtmp/btmp file
		DumpUtmp(File, Follow)
	} else if arg == "monitor" { // login/logout monitor
		Monitor(File, UseEUID)
	} else { // show error and exit if command is unknown
		log.Fatalf("error: unknown command '%s' (run with --help option)\n", arg)
	}
} // func main()

// Show active users from utmp/wtmp/btmp file
func ShowUsers(fname string, useEUID bool) {
	users, err := utmp.GetUsers(fname, useEUID)
	if err != nil {
		log.Fatalf("fatal: can't read utmp/wtmp/btmp file: %v\n", err)
	}

	for _, u := range users {
		u.Print(os.Stdout)
	}
}

// Show Full user info
func ShowUser(fname, username string, useEUID bool) {
	users, err := utmp.GetUsers(fname, useEUID)
	if err != nil {
		log.Fatalf("fatal: can't read utmp/wtmp/btmp file: %v\n", err)
	}

	li, err := users.GetLoginInfo(username)
	if err != nil {
		log.Fatalf("fatal: %v\n", err)
	}

	// Repack utmp.LoginInfo to dto.User
	u := dto.User{
		Name:        li.Name,
		UID:         li.UID,
		GID:         li.GID,
		DisplayName: li.DisplayName,
		HomeDir:     li.HomeDir,
		Groups:      li.Groups,
		LogonType:   dto.LogonType[li.Type],
		LogonTime:   li.Time,
		Logons:      li.Logons}

	// Encode full user info to JSON
	data, err := json.MarshalIndent(&u, "", "  ")
	if err != nil {
		log.Fatalf("fatal: json.Marshal():", err)
	}

	fmt.Println(string(data))
}

// Show logged user statistics (JSON)
func ShowUsersStat(fname string, useEUID bool) {
	users, err := utmp.GetUsers(fname, useEUID)
	if err != nil {
		log.Fatalf("fatal: can't read utmp/wtmp/btmp file: %v\n", err)
	}

	// get logged user statistics
	us := users.GetLoginStat()

	stat := dto.UsersStat{
		Total:      us.Total,
		LocalX:     us.LocalX,
		Local:      us.Local,
		RemoteX:    us.RemoteX,
		Remote:     us.Remote,
		Unknown:    us.Unknown,
		LocalRoot:  us.LocalRoot,
		RemoteRoot: us.RemoteRoot,
		Active:     us.Active.Name}

	// Encode statistics to JSON
	data, err := json.MarshalIndent(&stat, "", "  ")
	if err != nil {
		log.Fatalf("fatal: json.Marshal():", err)
	}

	fmt.Println(string(data))
}

// Dump utmp/wtmp/btmp file as plain text
func DumpUtmp(fname string, follow bool) {
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("fatal: can't open utmp/wtmp/btmp file: %v\n", err)
	}
	defer f.Close()

Loop:
	for {
		var u utmp.Utmp
		err = utmp.Read(f, &u)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				log.Fatalf(`fatal: read "%s": %v`, fname, err)
			}

			if !follow {
				break
			}

			select {
			case <-time.After(FOLLOW_INTERVAL):
			case <-signal.CtrlC:
				break Loop
			}
			continue
		}

		u.Print(os.Stdout)
	} // for
}

// Login/logout monitor
func Monitor(fname string, useEUID bool) {
	l, err := utmp.NewLogin(fname, useEUID)
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}

Loop:
	for {
		select {
		case evt := <-l.C():
			if len(evt.Login) != 0 {
				fmt.Printf(evt.Time.Format("2006-01-02 15:04:05"))
				fmt.Printf(" login:")
				for _, ut := range evt.Login {
					fmt.Printf(" %s[%s]", ut.User, ut.TTY)
				}
				if evt.Stat.Active != nil {
					fmt.Printf(" active=%s", evt.Stat.Active.Name)
				}
				fmt.Println()
			}

			if len(evt.Logout) != 0 {
				fmt.Printf(evt.Time.Format("2006-01-02 15:04:05"))
				fmt.Printf(" logout:")
				for _, ut := range evt.Logout {
					fmt.Printf(" %s[%s]", ut.User, ut.TTY)
				}
				if evt.Stat.Active != nil {
					fmt.Printf(" active=%s", evt.Stat.Active.Name)
				}
				fmt.Println()
			}

		case <-signal.CtrlC:
			break Loop
		}
	}
	l.Close()
}

// EOF: "gousers.go"
