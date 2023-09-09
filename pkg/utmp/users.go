// File: "users.go"

package utmp

import (
	"errors"
	"io"
	"net"
	"os"
	"regexp"
	"sort"
	"time"
)

// Структура описания пользователя системы на основе анализа Utmp записей.
// User structure delivered from Utmp (as-is + updated Name by EUID(PID)).
type User struct {
	Name string    // Username is the login name
	PID  uint32    // Process ID
	TTY  string    // TTY device
	Host string    // Login from
	IP   net.IP    // IPv4 address
	SID  int32     // Session ID
	ID   string    // Terminal name suffix
	Time time.Time // Time
}

// Список пользователей в системе на основе `utmp` файла.
type Users []*User

// Вспомогательная структура и интерфейсы для сортировки списка пользователей
// по времени входа в систему.
// UsersByTime implements sort.Interface for []*User based on the Time field
type UsersByTime Users

func (u UsersByTime) Len() int           { return len(u) }
func (u UsersByTime) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u UsersByTime) Less(i, j int) bool { return u[i].Time.Before(u[j].Time) }

// Определить тип входа пользователя по данным из `utmp` файла.
// Get user logon type (0...4).
func (u *User) LoginType() LoginType {
	reX := regexp.MustCompile("^:[0-9]+$") // user logged to X
	reRDP := regexp.MustCompile(XRDP_CMD)  // user logged by XRDP
	msX := reX.MatchString
	msRDP := reRDP.MatchString

	t := UNKNOWN
	if msX(u.Host) || msX(u.ID) || msX(u.TTY) { // e.g. ":1"
		if u.IP.Equal(net.IP{}) { // IP is empty
			t = LOCAL_X
			cmd, err := GetCmdline(u.PID)
			if err == nil && msRDP(cmd) {
				t = REMOTE_X // XRDP
			}
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

// Чтение utmp файла и формирования списка пользователей системы,
// фабричная функция для типа `Users`.
// (fname - путь к файлу utmp, обычно "/var/run/utmp").
// Get users currently logged in to the current host (fname - path to utmp file).
func GetUsers(fname string, useEUID bool) (Users, error) {
	if fname == "" {
		fname = DefaultFile
	}

	// Open utmp/wtmp/btmp file
	f, err := os.Open(fname)
	if err != nil {
		return Users{}, err // can't open file
	}
	defer f.Close()

	// инициализировать множества пользователей в системе
	base := make(map[UserTTY]*User)
	pbase := make(map[TTYPID]*User)
	ibase := make(map[TTYID]*User)

	// Read utmp/wtmp/btmp file
	for {
		var u Utmp
		err = Read(f, &u)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return Users{}, err
		}

		Type := int(u.Type)
		if Type == BOOT_TIME { // type 2
			base = make(map[UserTTY]*User)
			pbase = make(map[TTYPID]*User)
			ibase = make(map[TTYID]*User)
		} else if Type == USER_PROCESS || Type == DEAD_PROCESS { // type 7 or 8
			user := Str(u.User[:])
			pid := PID(u.PID)
			tty := Str(u.Line[:])
			id := Str(u.ID[:])

			ut := UserTTY{user, tty}
			tp := TTYPID{tty, pid}
			ti := TTYID{tty, id}

			p, ok := base[ut]

			if Type == USER_PROCESS { // user login
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

				Type := nu.LoginType()
				if Type == LOCAL && useEUID { // FIXME: some magic condition
					// Get real username by effective UID(pid)
					user, err := GetUserByPID(pid)
					if err == nil {
						nu.Name = user
					} else {
						// Do not show error (may read wtmp/btmp)
						// log.Printf("error: %v", err)
					}
				}

				if ok {
					if nu.Time.After(p.Time) {
						base[ut] = &nu // update base
						pbase[tp] = &nu
						ibase[ti] = &nu
					}
				} else {
					base[ut] = &nu // add to base
					pbase[tp] = &nu
					ibase[ti] = &nu
				}
			} else { // Type == DEAD_PROCESS => user logout
				if user == "" {
					// logout record in wtmp with User=""
					if u, ok := pbase[tp]; ok { // find logged TTY+PID
						ut.User = u.Name
					} else if u, ok := ibase[ti]; ok { // find logged TTY+ID
						ut.User = u.Name
					}
				}

				// delete from base
				delete(base, ut)
				delete(pbase, tp)
				delete(ibase, ti)
			}
		}
	} // for

	// Transform map to slice
	users := make(Users, 0, len(base))
	for _, u := range base {
		users = append(users, u)
	}

	// Sort by Time
	sort.Sort(UsersByTime(users))
	return users, nil
} // func UsersRead()

// Get user logon info by username
func (users Users) GetUserLogin(name string) (ul UserLogin) {
	for _, u := range users {
		if u.Name == name {
			ul.Logons++ // count number of logons
			if t := u.LoginType(); ul.Type < t {
				ul.Type = t // find max
				ul.Time = u.Time
			}
		}
	}
	return ul
}

// Вернуть полную информацию о пользователе в системе.
// Fill full user information.
func (users Users) GetLoginInfo(name string) (*LoginInfo, error) {
	info, err := GetUserInfo(name)
	if err != nil {
		return nil, err
	}
	ul := users.GetUserLogin(name)
	return &LoginInfo{
		UserInfo:  *info,
		UserLogin: ul}, nil
}

// Get logged user statistics
func (users Users) GetLoginStat() LoginStat {
	total := make(map[string]int)   // total logged users "Local + Remote + root"
	localX := make(map[string]int)  // users logged in X session
	local := make(map[string]int)   // local logged users (excluding root)
	remoteX := make(map[string]int) // remote users logged in X/xrdp/vnc
	remote := make(map[string]int)  // remote logged users (excluding root)
	unknown := make(map[string]int) // unknown logged users (must be empty)
	localRoot := false              // local root logged
	remoteRoot := false             // remote root logged
	user := (*User)(nil)            // main active user on host or nil
	Type := UNKNOWN                 // type of active user
	var active *LoginInfo           // main (active) user

	for _, u := range users {
		total[u.Name]++
		t := u.LoginType() // determinate user type

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
				user, Type = u, t
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

			if user == nil || Type <= t {
				user, Type = u, t
			}
		}
	} // for

	if user != nil {
		active, _ = users.GetLoginInfo(user.Name)
	}

	// Return result
	return LoginStat{
		Total:      len(total),
		LocalX:     len(localX),
		Local:      len(local),
		RemoteX:    len(remoteX),
		Remote:     len(remote),
		Unknown:    len(unknown),
		LocalRoot:  localRoot,
		RemoteRoot: remoteRoot,
		Active:     active}
}

// EOF: "users.go"
