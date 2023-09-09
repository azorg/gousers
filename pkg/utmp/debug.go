// utmp
// File: "debug.go"

package utmp

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

// Отладочная печать структуры `LoginInfo` в виде JSON.
// Debug print `UserInfo` as JSON.
func (u *LoginInfo) Print(f *os.File) {
	data, err := json.MarshalIndent(u, "", "  ") // human indent
	if err != nil {
		log.Printf("error: suddenly json.Marshal(): %v", err)
	}
	fmt.Fprintf(f, "%s\n", string(data))
}

// Отладочная печать структуры `User`.
// Debug print `User`.
func (u *User) Print(f *os.File) {
	fmt.Fprint(f, u.Time.Format("2006-01-02 15:04:05"))
	if u.Name != "" {
		fmt.Fprint(f, " Name='", u.Name, "'")
	}
	if u.TTY != "" {
		fmt.Fprint(f, " TTY='", u.TTY, "'")
	}
	if u.ID != "" {
		fmt.Fprint(f, " ID='", u.ID, "'")
	}

	fmt.Fprint(f, " PID=", u.PID)

	cmd, err := GetCmdline(u.PID)
	if err == nil {
		fmt.Fprint(f, " Cmd='", cmd, "'")
	}

	if u.Host != "" {
		fmt.Fprint(f, " Host='", u.Host, "'")
	}
	if !u.IP.Equal(net.IP{}) {
		fmt.Fprint(f, " IP=", u.IP)
	}
	if u.SID != 0 {
		fmt.Fprint(f, " SID=", u.SID)
	}
	fmt.Fprintln(f)
}

// Отладочная печать структуры `Utmp`.
// Debug print `Utmp`.
func (u *Utmp) Print(f *os.File) {
	t := Time(u.TV)
	fmt.Fprint(f, t.Format("2006-01-02 15:04:05"))

	Type := int(u.Type)
	fmt.Fprintf(f, " #%d %10s", Type, TypeString[Type])

	if u.Type == BOOT_TIME { // reboot
		if user := Str(u.User[:]); user != "" {
			fmt.Fprint(f, " User='", user, "'")
		}

		if host := Str(u.Host[:]); host != "" {
			fmt.Fprint(f, " Kernel='", host, "'")
		}
	} else if u.Type == RUN_LVL { // run level
		fmt.Fprint(f, " RL=", RunLvl(u.PID))
	} else {
		user := Str(u.User[:])

		if user != "" {
			fmt.Fprint(f, " User='", user, "'")
		}

		if tty := Str(u.Line[:]); tty != "" {
			fmt.Fprint(f, " TTY='", tty, "'")
		}

		if id := Str(u.ID[:]); id != "" {
			fmt.Fprint(f, " ID='", id, "'")
		}

		pid := PID(u.PID)
		if pid != 0 {
			fmt.Fprint(f, " PID=", pid)
		}

		euid, err := GetEUID(pid)
		if err == nil {
			fmt.Fprint(f, " EUID=", euid)
		}

		if host := Str(u.Host[:]); host != "" {
			fmt.Fprint(f, " Host='", host, "'")
		}

		if ip := IPv4(u.AddrV6); !ip.Equal(net.IP{}) {
			fmt.Fprint(f, " IP=", ip)
		}

		cmd, err := GetCmdline(pid)
		if err == nil {
			fmt.Fprint(f, " Cmd='", cmd, "'")
		}
	}

	if (u.Exit.Termination | u.Exit.Exit) != 0 {
		fmt.Fprint(f, " Term/Exit=", u.Exit.Termination, "/", u.Exit.Exit)
	}

	if u.Session != 0 {
		fmt.Fprint(f, " SID=", u.Session)
	}

	fmt.Fprintln(f)
}

// EOF: "debug.go"
