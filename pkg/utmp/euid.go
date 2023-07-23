// File "euid.go"

package utmp

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Get EUID by PID
func GetEUID(pid uint32) (int, error) {
	status := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(status)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fds := strings.Fields(scanner.Text())
		// man proc, look /proc/pid/status
		// line: "Uid: real, effective, saved, filesystem"
		if len(fds) >= 5 && fds[0] == "Uid:" {
			return strconv.Atoi(fds[2]) // euid, err
		}
	}

	return 0, fmt.Errorf(`can't find "^Uid: " in %s`, file)
}

// EOF: "euid.go"
