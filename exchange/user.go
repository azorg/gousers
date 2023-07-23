// File: "user.go"

package exchange

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
// Поле `Logons` указывает общее число входой пользовтаеля в систему (число
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

// EOF: "user.go"
