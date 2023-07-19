// File "utmp.go"

package utmp

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// Default file to read
var DefaultFile = "/var/run/utmp"

// Values for Type field
const (
	EMPTY         = 0 // Record does not contain valid info (unkonwn on Linux)
	RUN_LVL       = 1 // Change in system run-level (see `man 1 init`)
	BOOT_TIME     = 2 // Time of system boot (in `TV`)
	NEW_TIME      = 3 // Time after system clock change (in `TV`)
	OLD_TIME      = 4 // Time before system clock change (in `TV`)
	INIT_PROCESS  = 5 // Process spawned by init
	LOGIN_PROCESS = 6 // Session leader process for user login
	USER_PROCESS  = 7 // Normal process
	DEAD_PROCESS  = 8 // Terminated process
	ACCOUNTING    = 9 // Not implemented
)

// Types as test
var TypeString = [...]string{
	"EMPTY",      // 0
	"RUN_LVL",    // 1
	"REBOOT",     // 2
	"NEW_TIME",   // 3
	"OLD_TIME",   // 4
	"INIT_PROC",  // 5
	"LOGIN_PROC", // 6
	"USER_PROC",  // 7
	"DEAD_PROC",  // 8
	"ACCOUNTING", // 9
} // TypeString

// Magic sizes of Utmp fields
const (
	LINESIZE = 32
	NAMESIZE = 32
	HOSTSIZE = 256
)

// utmp struct for 64-bit platforms
type Utmp struct {
	Type        int16          // Type of record
	Pad0_unused [2]byte        //
	PID         [4]byte        // PID of login process
	Line        [LINESIZE]int8 // Device name of tty - "/dev/"
	ID          [4]int8        // Terminal name suffix, or inittab ID
	User        [NAMESIZE]int8 // Username
	Host        [HOSTSIZE]int8 // Hostname for remote login, or kernel ver. for run-level messages
	Exit        ExitStatus     // Exit status of a process marked as DEAD_PROCESS, not used by Linux init
	Session     int32          // Session ID (getsid(2)) used for windowing
	TV          TimeVal        // Time entry was made
	AddrV6      [4]int32       // IP address of remote host (IPv4 address uses just AddrV6[0])
	Pad1_unused [20]int8       // Reserved for future use
} // type Utmp

// Type of exit status
type ExitStatus struct {
	Termination int16 // Process termination status
	Exit        int16 // Process exit status
} // type ExitStatus

// Type of time entry
type TimeVal struct {
	Sec  int32 // Seconds
	Usec int32 // Microseconds
} // type TimeVal

// Read one record of Utmp from binary file
func Read(file io.Reader, utmp *Utmp) error {
	return binary.Read(file, binary.LittleEndian, utmp)
} // func Read()

// Convert Utmp chars to string
func Str(src []int8) string {
	b := make([]byte, 0, len(src))
	for _, v := range src {
		if v == 0 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
} // func Str()

// Convert time stamp to Unix time
func Time(tv TimeVal) time.Time {
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000) // usec -> nsec
} // func Time()

// Get PID from Utmp
func PID(pid [4]byte) uint32 {
	return binary.LittleEndian.Uint32(pid[:])
} // func PID()

// Get RunLevel from Utmp
func RunLvl(pid [4]byte) string {
	b := pid[0]
	if b > 0x20 {
		return fmt.Sprintf("%c", b)
	} else {
		return fmt.Sprintf("0x%02X", b)
	}
} // func RunLvl()

// Get IPv4 address from AddrV6
func IPv4(addrV6 [4]int32) (IP string) {
	ip := uint32(addrV6[0])
	if ip != 0 {
		b3 := (ip >> 24) & 0xFF
		b2 := (ip >> 16) & 0xFF
		b1 := (ip >> 8) & 0xFF
		b0 := (ip >> 0) & 0xFF
		IP = fmt.Sprintf("%d.%d.%d.%d", b3, b2, b1, b0)
	}
	return
} // func IPv4

// Debug print Utmp
func Print(f *os.File, u Utmp) {
	t := Time(u.TV)
	fmt.Fprint(f, t.Format("2006-02-01 15:04:05"))

	Type := int(u.Type)
	fmt.Fprintf(f, " #%d %10s", Type, TypeString[Type])

	if u.Type == BOOT_TIME { // reboot
		if user := Str(u.User[:]); user != "" {
			fmt.Fprint(f, " User='", user, "'")
		}

		if host := Str(u.Host[:]); host != "" {
			fmt.Fprint(f, " Kernel='", host, "'")
		}

		fmt.Fprint(f, " SID=", u.Session)
		fmt.Fprintln(f)
		return
	}

	if u.Type == RUN_LVL { // run level
		fmt.Fprint(f, " RL=", RunLvl(u.PID))
		fmt.Fprintln(f)
		return
	}

	if user := Str(u.User[:]); user != "" {
		fmt.Fprint(f, " User='", user, "'")
	}

	fmt.Fprint(f, " TTY='", Str(u.Line[:]), "'")
	fmt.Fprint(f, " ID='", Str(u.ID[:]), "'")
	fmt.Fprint(f, " PID=", PID(u.PID))

	if host := Str(u.Host[:]); host != "" {
		fmt.Fprint(f, " Host='", host, "'")
	}

	if ip := IPv4(u.AddrV6); ip != "" {
		fmt.Fprint(f, " IP=", ip)
	}

	fmt.Fprint(f, " Term/Exit=", u.Exit.Termination, "/", u.Exit.Exit)
	fmt.Fprint(f, " SID=", u.Session)

	fmt.Fprintln(f)
} // func Print()

// EOF: "utmp.go"
