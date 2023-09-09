// File: "api.go"

package utmp

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Файл для чтения по умолчанию.
// Default file to read.
var DefaultFile = DEFAULT_FILE

// Типы пользователей.
// Type of logged user (5 types: 0-4).
var LoginTypeStr = [...]string{"", "remote", "remote_x", "local", "local_x"}

type LoginType int

const (
	UNKNOWN LoginType = iota // тип пользователя не определен (авария)

	REMOTE   // удаленный пользователь (ssh)
	REMOTE_X // удаленный пользователь графического сеанса (FIXME: НЕ реализовано)
	LOCAL    // локальный пользователь (вход через login/sudo)
	LOCAL_X  // локальный пользователь графического сеанса (вход через Desktop manager)
)

// Стандартные данные пользователя, предоставляемые структурой `os/user.User`.
// User information delivered from `os/user.User`.
type UserInfo struct {
	Name        string // Username is the login name
	UID         string // User ID
	GID         string // Primary group ID
	DisplayName string // User display name (may be empty)
	HomeDir     string // User's home directory
	Groups      string // Groups that the user is a member of (CSV)
}

// Метрики пользователя на основе данных из utmp файла.
// Logged user metrics.
type UserLogin struct {
	Type   LoginType // Тип входа пользователя: 0..4: unknown..local_x
	Time   time.Time // Последнее время входа пользователя
	Logons int       // Число входов пользователя в систему
}

// Структура полной информации о пользователе в системе.
// Full user information.
type LoginInfo struct {
	UserInfo
	UserLogin
}

// Статистика входов пользователей.
// Logged user statistics.
type LoginStat struct {
	Total      int        // Total logged users "Local + Remote + root"
	LocalX     int        // Number of users logged in X session (excluding root)
	Local      int        // Number of local users (excluding root)
	RemoteX    int        // Number of remote users logged in X/xrdp/vnc (excluding root)
	Remote     int        // Number of remote users (excluding root)
	Unknown    int        // Total number of unknown logged users (must be 0)
	LocalRoot  bool       // Local root logged
	RemoteRoot bool       // Remote root logged
	Active     *LoginInfo // Information about active user or nil
}

// Вспомагательная структура для сохрнения имени пользователя и терминала.
type UserTTY struct {
	User string // User name
	TTY  string // TTY device
}

// Вспомагательная структура для сохрнения имени терминала и PID
// (нужно для анализа wtmp).
type TTYPID struct {
	TTY string // TTY device
	PID uint32 // Process ID
}

// Вспомагательная структура для сохрнения имени терминала и ID
// (нужно для анализа wtmp в Astra Linux).
type TTYID struct {
	TTY string // TTY device
	ID  string // wtmp ID
}

// Структура события изменения `utmp` файла - входа/выхода пользователя.
// Event of utmp file changed.
type LoginEvent struct {
	// Время последнего обновления utmp файла
	Time time.Time

	// Имена пользователей вновь вошедших (с указанием терминала)
	Login []UserTTY

	// Имена пользователей только что вышедших (с указанием терминала)
	Logout []UserTTY

	// Полное описание пользователей в системе на данный момент
	Users []LoginInfo

	// Статистика пользователей, в т.ч. информация об активном пользователе сеанса
	Stat LoginStat
}

// Интерфейс класса Login
type Loginer interface {
	Close()                // Terminate
	C() <-chan LoginEvent  // Get event channel
	GetUsers() []LoginInfo // Get logded user information
	GetStat() LoginStat    // Get logged user statistics
}

// Класс для отслеживания событий входа/выхода пользователей
// и оперативного извлечения из памяти данных о текущем активном (основном)
// пользователе сеанса для работы службы контроля съёмных носителей.
type Login struct {
	// Все поля структуры "приватные".
	// Has unexported fields.
	fname    string               // полный путь к файлу utmp
	useEUID  bool                 // признак использования эффективного UID
	evtChan  chan LoginEvent      // канал для передачи событий изменения utmp
	watcher  *fsnotify.Watcher    // компонент fsnotify
	users    Users                // списко пользователей полученный из utmp
	logged   map[UserTTY]struct{} // перечень пользователей в системе с терминалами
	logins   []LoginInfo          // подробная информация о всех пользователях системы
	loginsMx sync.RWMutex         // мьютекс для защиты `logins`
	stat     LoginStat            // статистика пользователей
	statMx   sync.RWMutex         // мьютекс для защиты `stat`
	wg       sync.WaitGroup       // группа ожидания при завершении работы
}

// Фабричная функция для создания экземпляра класса (конструктор).
// (fname - полный путь к файлу utmp, например "/var/run/utmp" или ""
// - для использования файла по умолчанию).
func NewLogin(fname string, useEUID bool) (*Login, error) {
	if fname == "" {
		fname = DefaultFile
	}
	l := &Login{fname: fname, useEUID: useEUID}
	l.evtChan = make(chan LoginEvent)

	// Создать объект fsnotify.Watcher
	var err error
	l.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = l.watcher.Add(fname)
	if err != nil {
		return nil, err
	}

	// Инициировать пустое множество пользователей в системе
	l.logged = make(map[UserTTY]struct{})

	// Запустить горутину ожидания событий от объекта fsnotify.Watcher
	l.wg.Add(1)
	go watcherFn(l)

	// Дождаться завершения первого чтения utmp файла
	<-l.evtChan

	return l, nil
}

// Функция деинициализации (деструктор, освобождение ресурсов,
// закрытие канала событий, останов горутин).
func (l *Login) Close() {
	close(l.evtChan)
	l.watcher.Close()
	l.wg.Wait()
}

// Функция/метод получения (не буферизированного) канала для получения событий.
func (l *Login) C() <-chan LoginEvent {
	return l.evtChan
}

// Функция/метод получения (из памяти) полной информация
// обо всех пользователях в системе
func (l *Login) GetUsers() []LoginInfo {
	l.loginsMx.RLock()
	defer l.loginsMx.RUnlock()
	logins := make([]LoginInfo, len(l.logins))
	copy(logins, l.logins)
	return logins
}

// Функция/метод получения (из памяти) полной информация о текущем (активном)
// пользователе сеанса.
func (l *Login) GetStat() LoginStat {
	l.statMx.RLock()
	defer l.statMx.RUnlock()
	stat := l.stat
	return stat
}

// EOF: "api.go"
