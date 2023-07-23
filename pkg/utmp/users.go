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
	"strconv"
	"strings"
	"time"
)

// User structure delivered from Utmp (as-is + updated Name by EUID(PID))
type User struct {
	Name string    `json:"name"`           // Username is the login name
	PID  uint32    `json:"pid,omitempty"`  // Process ID
	TTY  string    `json:"tty,omitempty"`  // TTY device
	Host string    `json:"host,omitempty"` // Login from
	IP   net.IP    `json:"ip,omitempty"`   // IPv4 address
	SID  int32     `json:"sid,omitempty"`  // Session ID
	ID   string    `json:"id,omitempty"`   // Terminal name suffix
	Time time.Time `json:"time,omitempty"` // Time
}

// Type of logged user (5 types)
var UserType = [...]string{"", "remote", "remote_x", "local", "local_x"}

const (
	UNKNOWN = iota
	REMOTE
	REMOTE_X
	LOCAL
	LOCAL_X
)

// Logged user metrics
type UserLogon struct {
	Type   int       `json:"type,omitempty"`   // Type of logon: 0..5: unknown..local_x
	Time   time.Time `json:"time,omitempty"`   // Last logon time
	Logons int       `json:"logons,omitempty"` // Number of user logons
}

// User information delivered from user.User
type UserInfo struct {
	Name        string `json:"name"`                   // Username is the login name
	UID         string `json:"uid,omitempty"`          // User ID
	GID         string `json:"gid,omitempty"`          // Primary group ID
	DisplayName string `json:"display_name,omitempty"` // User display name (may be empty)
	HomeDir     string `json:"home_dir,omitempty"`     // User's home directory
	Groups      string `json:"groups,omitempty"`       // Groups that the user is a member of (CSV)
}

// Full user information (equivalent of export.User)
type UserFull struct {
	UserInfo
	UserLogon
}

// Logged user statistics
type UsersStat struct {
	Total      int       `json:"total"`                 // Total logged users "Local + Remote + root"
	LocalX     int       `json:"local_x"`               // Number of users logged in X session (excluding root)
	Local      int       `json:"local"`                 // Number of local users (excluding root)
	RemoteX    int       `json:"remote_x"`              // Number of remote users logged in X/xrdp/vnc (excluding root)
	Remote     int       `json:"remote"`                // Number of remote users (excluding root)
	Unknown    int       `json:"unknown,omitempty"`     // Total number of unknown logged users (must be 0)
	LocalRoot  bool      `json:"local_root,omitempty"`  // Local root logged
	RemoteRoot bool      `json:"remote_root,omitempty"` // Remote root logged
	User       *UserFull `json:"user,omitempty"`        // Information about active user or nil
}

type userTTY struct {
	User string // User name
	TTY  string // TTY device
}

// UsersByTime implements sort.Interface for []*User based on the Time field
type UsersByTime []*User

func (u UsersByTime) Len() int           { return len(u) }
func (u UsersByTime) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u UsersByTime) Less(i, j int) bool { return u[i].Time.Before(u[j].Time) }

// Get effective username by PID
func GetUserByPID(pid uint32) (string, error) {
	euid, err := GetEUID(pid)
	if err != nil {
		return "", err
	}

	u, err := user.LookupId(strconv.Itoa(euid))
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

// Get users currently logged in to the current host (fname - path to utmp file)
func Users(fname string, useEUID bool) ([]*User, error) {
	if fname == "" {
		fname = DefaultFile
	}

	// Open utmp/wtmp/btmp file
	f, err := os.Open(fname)
	if err != nil {
		return []*User{}, err // can't open file
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
			user := Str(u.User[:])
			pid := PID(u.PID)
			tty := Str(u.Line[:])
			ut := userTTY{user, tty}
			p, ok := base[ut]

			if Type == USER_PROCESS { // user login
				if useEUID {
					// Get real username by effective UID(pid)
					realUser, err := GetUserByPID(pid)
					if err == nil {
						user = realUser
					} else {
						// Do not show error (may read wtmp/btmp)
						// log.Printf("error: %v", err)
					}
				}

				nu := User{
					Name: user,
					PID:  pid,
					TTY:  tty,
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

// Get user logon type (0...5)
func GetUserType(u *User) int {
	reX := regexp.MustCompile("^:[0-9]+$") // user logged to X
	msX := reX.MatchString

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
	return t
}

// Get user logon info by username
func GetUserLogon(users []*User, name string) (ul UserLogon) {
	for _, u := range users {
		if u.Name == name {
			ul.Logons++ // count number of logons
			if t := GetUserType(u); ul.Type < t {
				ul.Type = t // fix max
				ul.Time = u.Time
			}
		}
	}
	return ul
}

// Get user info by username delivered from user.User
func GetUserInfo(username string) (info *UserInfo, err error) {
	u, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	info = &UserInfo{
		UID:         u.Uid,
		GID:         u.Gid,
		Name:        u.Username,
		DisplayName: u.Name,
		HomeDir:     u.HomeDir}

	// Find groups that the user is a member of
	gids, err := u.GroupIds()
	if err != nil {
		return nil, err
	}

	var groups []string
	for _, gid := range gids {
		grp, err := user.LookupGroupId(gid)
		if err != nil {
			return info, err
		}
		groups = append(groups, grp.Name)
	}

	info.Groups = strings.Join(groups, ",")
	return info, nil
} // func GetUserInfo()

// Fill full user information
func GetUserFull(users []*User, name string) (*UserFull, error) {
	info, err := GetUserInfo(name)
	if err != nil {
		return nil, err
	}
	ul := GetUserLogon(users, name)
	return &UserFull{
		UserInfo:  *info,
		UserLogon: ul}, nil
}

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

	for _, u := range users {
		total[u.Name]++
		t := GetUserType(u) // determinate user type

		if u.Name == "root" {
			switch t {
			case LOCAL_X, LOCAL:
				localRoot = true
			case REMOTE_X, REMOTE:
				remoteRoot = true
			default: // UNKNOWN
				localRoot = true // unknown root as local
				unknown[u.Name]++
			} // switch

			if user == nil || user.Name == "root" {
				user, userType = u, t
			}
		} else { // regular user
			switch t {
			case LOCAL_X:
				localX[u.Name]++
			case LOCAL:
				local[u.Name]++
			case REMOTE_X:
				remoteX[u.Name]++
			case REMOTE:
				remote[u.Name]++
			default: // UNKNOWN
				unknown[u.Name]++
			} // switch

			if user == nil || userType <= t {
				user, userType = u, t
			}
		}
	} // for

	full, _ := GetUserFull(users, user.Name)

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
		User:       full}
} // func GetUsersStat()

// Debug print User
func UserPrint(f *os.File, u *User) {
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

// EOF: "users.go"
