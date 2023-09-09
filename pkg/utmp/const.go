// utmp
// File: "const.go"

package utmp

// Файл для чтения по умолчанию (обычно /var/run/utmp или /var/log/wtmp).
// Default file to read.
const DEFAULT_FILE = "/var/log/wtmp"

// Название XRDP программы для определения удалённых X пользователей.
// XRDP programm for detect remote X users.
const XRDP_CMD = "xrdp-sesman"

// EOF: "const.go"
