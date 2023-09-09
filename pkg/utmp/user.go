// File: "user.go"

package utmp

import (
	"os/user"
	"strconv"
	"strings"
)

// Получить эффективное имя пользователя по Process ID.
// Get effective username by PID.
func GetUserByPID(pid uint32) (username string, err error) {
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

// Получить информацию о пользователе из стандартной структуры `os/user.User`.
// Get user info by username delivered from `os/user.User`
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
}

// EOF: "user.go"
