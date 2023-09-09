// File: "login.go"

package utmp

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

// Поиск вновь вошедших, только что вышедших пользовтаелей.
// Find login/logout users
func (l *Login) findLoginLogout() (login, logout []UserTTY) {
	m := make(map[UserTTY]struct{})

	// Найти вновь вошедших (find login)
	for _, u := range l.users {
		ut := UserTTY{u.Name, u.TTY}
		if _, ok := l.logged[ut]; !ok {
			l.logged[ut] = struct{}{}
			login = append(login, ut)
		}
		m[ut] = struct{}{}
	}

	// Найти только что вышедших (find logout)
	for ut, _ := range l.logged {
		if _, ok := m[ut]; !ok {
			delete(l.logged, ut)
			logout = append(logout, ut)
		}
	}
	return login, logout
}

// Прочитать utmp файл, сохранить/распаковать данные, послать событие.
// Read utmp file, save/parse data, send event.
func (l *Login) readUtmp() {
	// Получить время обновления utmp файла
	Stat, err := os.Stat(l.fname)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	modTime := Stat.ModTime()

	// Прочитать (обновленный) utmp файл
	l.users, err = GetUsers(l.fname, l.useEUID)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	// Определить кто вошел/кто вышел (find login/logout users)
	login, logout := l.findLoginLogout()

	// Получить полную информацию о всех пользователях в системе (logins).
	// Результирующий список сортирован по времени.
	// Get full info about logged users (sorted by time)
	logins := []LoginInfo{}
	umap := make(map[string]int) // индекс пользователя в списке по имени
	cnt := 0
	for _, u := range l.users {
		info, err := l.users.GetLoginInfo(u.Name)
		if err != nil {
			log.Printf("error: %v", err)
			return
		}
		ix, ok := umap[info.Name]
		if ok {
			logins[ix] = *info // update (users sorted by time)
		} else {
			logins = append(logins, *info)
			umap[info.Name] = cnt
			cnt++
		}
	}

	// Сохранить в памяти список всех пользователей системы
	l.loginsMx.Lock()
	l.logins = make([]LoginInfo, len(logins))
	copy(l.logins, logins)
	l.loginsMx.Unlock()

	// Получить статистику и сохранить в памяти
	stat := l.users.GetLoginStat()
	l.statMx.Lock()
	l.stat = stat
	l.statMx.Unlock()

	// Write event to channel
	l.evtChan <- LoginEvent{
		Time:   modTime,
		Login:  login,
		Logout: logout,
		Users:  logins,
		Stat:   stat}
}

// Горутина ожидания событий fsnotify,
// fsnotify goroutine.
func watcherFn(l *Login) {
	l.readUtmp() // первый раз прочитать utmp не ожидая события

For:
	for {
		select {
		case evt, ok := <-l.watcher.Events:
			if !ok {
				break For
			}
			//log.Print("fsnotify: ", evt)
			if evt.Has(fsnotify.Write) {
				l.readUtmp() // нас интересует только события обновления файла
			}
		case err, ok := <-l.watcher.Errors:
			if !ok {
				break For
			}
			log.Print("error:", err)
		} // select
	} // for
	l.wg.Done()
}

// EOF: "login.go"
