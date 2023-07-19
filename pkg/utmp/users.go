// File: "users.go"

package utmp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

// User structure delivered from Utmp
type User struct {
	Name string    // User name
	PID  uint32    // Process ID
	TTY  string    // TTY device
	Host string    // Login from
	IP   string    // IPv4 address
	SID  int32     // Session ID
	ID   string    // Terminal name suffix
	Time time.Time // Time
} // type User

type userTTY struct {
	User string // User name
	TTY  string // TTY device
} // type userTTY

// UsersByTime implements sort.Interface for []*User based on the Time field
type UsersByTime []*User

func (u UsersByTime) Len() int           { return len(u) }
func (u UsersByTime) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u UsersByTime) Less(i, j int) bool { return u[i].Time.Before(u[j].Time) }

// Get users currently logged in to the current host
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

// Debug print User
func UserPrint(f *os.File, u *User) {
	fmt.Fprint(f, u.Time.Format("2006-02-01 15:04:05"))
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
	if u.IP != "" {
		fmt.Fprint(f, " IP=", u.Host)
	}
	fmt.Fprint(f, " SID=", u.SID)
	fmt.Fprintln(f)
} // func UserPrint()

// EOF: "users.go"
