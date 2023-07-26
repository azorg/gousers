// File: "notify.go"

package utmp

import (
	"gousers/pkg/vsnotify"
	"log"
	"os"
	"sync"
	"time"
)

var DEBUG = false // debug output to log on terminate

// Event of utmp file changed
type Event struct {
	Time   time.Time  // Time of utmp file update
	Login  []UserTTY  // Login users
	Logout []UserTTY  // Logout users
	Users  []UserFull // Full information about all logged users
	Stat   UsersStat  // User statistics
}

type Notify struct {
	Update <-chan Event
	Cancel func()
}

// Create new utmp notifier
//
//	utmpFile - path to utmp file (/var/run/utmp)
//	useEUID - use EUID(PID) and find effective user
//	period - period of utmp file check
func NewNotify(utmpFile string, useEUID bool, period time.Duration) (
	*Notify, error) {

	stat, err := os.Stat(utmpFile)
	if err != nil {
		return nil, err
	}
	modTime := stat.ModTime()

	n, err := vsnotify.New(utmpFile, period)
	if err != nil {
		return nil, err
	}

	logged := make(map[UserTTY]struct{})
	update := make(chan Event)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		upd := func(time time.Time) bool {
			// Get all logged used from utmp file
			users, err := Users(utmpFile, useEUID)
			if err != nil {
				log.Printf("error: %v", err)
				return false
			}

			// Find login/logout users
			login, logout := findLoginLogout(logged, users)

			// Get full info about logged users
			list := make([]UserFull, 0, len(users))
			for _, u := range users {
				info, err := GetUserFull(users, u.Name)
				if err != nil {
					log.Printf("error: %v", err)
					return true
				}
				list = append(list, *info)
			}

			// Get users statistics
			stat := GetUsersStat(users)

			// Write event to channel
			update <- Event{Time: time, Login: login, Logout: logout, Users: list, Stat: stat}
			return true
		} // func upd()

		ok := upd(modTime)
		for ok {
			modTime, ok = <-n.Update
			if ok {
				ok = upd(modTime)
			}
		}

		close(update)
		wg.Done()
	}()

	cancel := func() {
		n.Cancel()
		wg.Wait()
		if DEBUG {
			log.Printf("utmpNotifier(%s) done", utmpFile)
		}
	}

	return &Notify{update, cancel}, nil
}

// Find login/logout users
func findLoginLogout(logged map[UserTTY]struct{}, users []*User) (
	login, logout []UserTTY) {
	m := make(map[UserTTY]struct{})

	// find login
	for _, u := range users {
		ut := UserTTY{u.Name, u.TTY}
		if _, ok := logged[ut]; !ok {
			logged[ut] = struct{}{}
			login = append(login, ut)
		}
		m[ut] = struct{}{}
	}

	// find logout
	for ut, _ := range logged {
		if _, ok := m[ut]; !ok {
			delete(logged, ut)
			logout = append(logout, ut)
		}
	}
	return // login, logout
}

// EOF: "notify.go"
