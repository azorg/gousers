// File: "users.go"

package utmp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"regexp"
	"sort"
	"time"
)

// User structure delivered from Utmp
type User struct {
	Name string    `json:"name,omitempty"` // User name
	PID  uint32    `json:"pid,omitempty"`  // Process ID
	TTY  string    `json:"tty,omitempty"`  // TTY device
	Host string    `json:"host,omitempty"` // Login from
	IP   net.IP    `json:"ip,omitempty"`   // IPv4 address
	SID  int32     `json:"sid,omitempty"`  // Session ID
	ID   string    `json:"id,omitempty"`   // Terminal name suffix
	Time time.Time `json:"time,omitempty"` // Time
} // type User

// Type of active logged user (5 types)
var UserType = [...]string{"", "remote", "remote_x", "local", "local_x"}

const (
	UNKNOWN = iota
	REMOTE
	REMOTE_X
	LOCAL
	LOCAL_X
)

// Logged user statistics
type UsersStat struct {
	Total      int    `json:"total"`                 // Total logged users "Local + Remote + root"
	LocalX     int    `json:"local_x"`               // Number of users logged in X session (excluding root)
	Local      int    `json:"local"`                 // Number of local users (excluding root)
	RemoteX    int    `json:"remote_x"`              // Number of remote users logged in X/xrdp/vnc (excluding root)
	Remote     int    `json:"remote"`                // Number of remote users (excluding root)
	Unknown    int    `json:"unknown,omitempty"`     // Total number of unknown logged users (must be 0)
	LocalRoot  bool   `json:"local_root,omitempty"`  // Local root logged
	RemoteRoot bool   `json:"remote_root,omitempty"` // Remote root logged
	User       *User  `json:"user,omitempty"`        // Main active user on host or nil
	UserType   string `json:"user_type,omitempty"`   // Type of active user
	UserLogons int    `json:"user_logons,omitempty"` // Number of active user logons
} // type UsersStat

type userTTY struct {
	User string // User name
	TTY  string // TTY device
} // type userTTY

// UsersByTime implements sort.Interface for []*User based on the Time field
type UsersByTime []*User

func (u UsersByTime) Len() int           { return len(u) }
func (u UsersByTime) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u UsersByTime) Less(i, j int) bool { return u[i].Time.Before(u[j].Time) }

// Get users currently logged in to the current host (fname - path to utmp file)
func Users(fname string) ([]*User, error) {
	if fname == "" {
		fname = DefaultFile
	}

	// Open utmp/wtmp/btmp file
	f, err := os.Open(fname)
	if err != nil {
		return []*User{}, err
	}
	defer f.Close()

	base := make(map[userTTY]*User)

	// Read utmp/wtmp/btmp file
	for {
		var u Utmp
		err = Read(f, &u)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			} else {
				return []*User{}, err
			}
		}

		Type := int(u.Type)
		if Type == USER_PROCESS || Type == DEAD_PROCESS { // type 7 or 8
			tty := Str(u.Line[:])
			user := Str(u.User[:])
			ut := userTTY{user, tty}
			p, ok := base[ut]

			if Type == USER_PROCESS { // user login
				nu := User{
					Name: Str(u.User[:]),
					PID:  PID(u.PID),
					TTY:  Str(u.Line[:]),
					Host: Str(u.Host[:]),
					IP:   IPv4(u.AddrV6),
					SID:  u.Session,
					ID:   Str(u.ID[:]),
					Time: Time(u.TV),
				}

				if ok {
					if nu.Time.After(p.Time) {
						base[ut] = &nu // update base
					}
				} else {
					base[ut] = &nu // add to base
				}
			} else { // Type == DEAD_PROCESS => user logout
				delete(base, ut) // delete from base
			}
		}
	} // for

	// Transform map to slice
	users := make([]*User, 0, len(base))
	for _, u := range base {
		users = append(users, u)
	}

	// Sort by Time
	sort.Sort(UsersByTime(users))
	return users, nil
} // func Users()

// Get logged user statistics
func GetUsersStat(users []*User) UsersStat {
	total := make(map[string]int)   // total logged users "Local + Remote + root"
	localX := make(map[string]int)  // users logged in X session
	local := make(map[string]int)   // local logged users (excluding root)
	remoteX := make(map[string]int) // remote users logged in X/xrdp/vnc
	remote := make(map[string]int)  // remote logged users (excluding root)
	unknown := make(map[string]int) // unknown logged users (must be empty)
	localRoot := false              // local root logged
	remoteRoot := false             // remote root logged
	user := (*User)(nil)            // main active user on host or nil
	userType := UNKNOWN             // type of active user

	reX := regexp.MustCompile("^:[0-9]+$") // user logged to X
	msX := reX.MatchString

	for _, u := range users {
		total[u.Name]++

		// Determinate user type
		t := UNKNOWN
		if msX(u.Host) || msX(u.ID) || msX(u.TTY) { // e.g. ":1"
			if u.IP.Equal(net.IP{}) { // IP is empty
				t = LOCAL_X
			} else {
				t = REMOTE_X // FIXME: bad code, fix it later
			}
		} else {
			if u.IP.Equal(net.IP{}) && u.Host == "" { // IP and Host is empty
				t = LOCAL
			} else {
				t = REMOTE
			}
		}

		if u.Name == "root" {
			if t == LOCAL_X || t == LOCAL {
				localRoot = true
			} else if t == REMOTE_X || t == REMOTE {
				remoteRoot = true
			} else { // t == UNKNOWN
				localRoot = true // unknown root as local
				unknown[u.Name]++
			}

			if user == nil {
				user, userType = u, t
			}
		} else { // regular user
			if t == LOCAL_X {
				localX[u.Name]++
			} else if t == LOCAL {
				local[u.Name]++
			} else if t == REMOTE_X {
				remoteX[u.Name]++
			} else if t == REMOTE {
				remote[u.Name]++
			} else { // t == UNKNOWN
				unknown[u.Name]++
			}

			if user == nil || userType <= t {
				user, userType = u, t
			}
		}
	} // for

	// Return result
	return UsersStat{
		Total:      len(total),
		LocalX:     len(localX),
		Local:      len(local),
		RemoteX:    len(remoteX),
		Remote:     len(remote),
		Unknown:    len(unknown),
		LocalRoot:  localRoot,
		RemoteRoot: remoteRoot,
		User:       user,
		UserType:   UserType[userType],
		UserLogons: total[user.Name]}
} // func GetUsersStat()

// Get active linux user as user.User by username (or current user)
func ActiveUser(name string) (*user.User, error) {
	if name != "" {
		return user.Lookup(name)
	} else {
		return user.Current()
	}
} // func ActiveUser()

// Debug print User
func UserPrint(f *os.File, u *User) {
	fmt.Fprint(f, u.Time.Format("2006-01-02 15:04:05"))
	if u.Name != "" {
		fmt.Fprint(f, " User='", u.Name, "'")
	}
	if u.TTY != "" {
		fmt.Fprint(f, " TTY='", u.TTY, "'")
	}
	if u.ID != "" {
		fmt.Fprint(f, " ID='", u.ID, "'")
	}
	fmt.Fprint(f, " PID=", u.PID)
	if u.Host != "" {
		fmt.Fprint(f, " Host='", u.Host, "'")
	}
	if !u.IP.Equal(net.IP{}) {
		fmt.Fprint(f, " IP=", u.IP)
	}
	fmt.Fprint(f, " SID=", u.SID)
	fmt.Fprintln(f)
} // func UserPrint()

// EOF: "users.go"
