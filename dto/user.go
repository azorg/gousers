// File: "user.go"

package dto

import "time"

// Type of logged user (5 types, first  - empty/unknown)
var LogonType = [...]string{"", "remote", "remote_x", "local", "local_x"}

// Описние пользователя (реплика user.User + перечень групп + тип/статистика входа).
//
// Поле `Name` содержит имя пользователя - username.
//
// Перечень групп, в которых состоит пользователь уканан в поле `Groups`,
// значения разделены запятыми.
//
// Описана стуктура пользователей сеанса с учётом параметров
// их входа в систему (удаленные пользователи, локальные и т.п.),
// см. поле `LogonType:`
//
//	`local_x` - локальный пользователь графической подсистемы;
//	`local` - локальный пользователь терминала;
//	`remote_x` - удаленный пользователель графической подсистемы;
//	`remote` -  удаленный пользователь (ssh).
//
// Поле `LogonTime` содержит время последнего входа данного пользователя в систему.
//
// Поле `Logons` указывает общее число входов пользовтаеля в систему (число
// окрытых сеансов X-window, число открытых виртуальных консолей и т.п.).
type User struct {
	Name        string    `json:"name"`                   // Username is the login name (unuq Security ID)
	UID         string    `json:"uid,omitempty"`          // User ID (decimal integer)
	GID         string    `json:"gid,omitempty"`          // Primary group ID (decimal integer)
	DisplayName string    `json:"display_name,omitempty"` // User display name (may be empty)
	HomeDir     string    `json:"home_dir,omitempty"`     // User's home directory
	Groups      string    `json:"groups,omitempty"`       // Groups that the user is a member of (CSV)
	LogonType   string    `json:"logon_type,omitempty"`   // Type of logon of user: remote, remote_x, local, local_x
	LogonTime   time.Time `json:"logon_time,omitempty"`   // Last logon time
	Logons      int       `json:"logons,omitempty"`       // Number of user logons (local+remote) >=1
}

// Logged user statistics.
// Описание статистики логинов.
// Поле Active соджержит имя "главного" пользователя системы.
type UsersStat struct {
	Total      int    `json:"total,omitempty"`       // Total logged users "Local + Remote + root"
	LocalX     int    `json:"local_x,omitempty"`     // Number of users logged in X session (excluding root)
	Local      int    `json:"local,omitempty"`       // Number of local users (excluding root)
	RemoteX    int    `json:"remote_x,omitempty"`    // Number of remote users logged in X/xrdp/vnc (excluding root)
	Remote     int    `json:"remote,omitempty"`      // Number of remote users (excluding root)
	Unknown    int    `json:"unknown,omitempty"`     // Total number of unknown logged users (must be 0)
	LocalRoot  bool   `json:"local_root,omitempty"`  // Local root logged
	RemoteRoot bool   `json:"remote_root,omitempty"` // Remote root logged
	Active     string `json:"active,omitempty"`      // Active user (or "")
}

// EOF: "user.go"
