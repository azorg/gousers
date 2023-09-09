// File "proc.go"

package utmp

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Получить эффективный User ID по Process ID.
// Get EUID by PID.
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

	return 0, fmt.Errorf(`can't find "^Uid: " in %s`, file.Name())
}

// Получить строку запуска процесса по Process ID.
// Get CmdLine by PID
func GetCmdline(pid uint32) (string, error) {
	file := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmd, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	cmd = bytes.TrimRight(cmd, string([]byte{0}))
	cmd = bytes.ReplaceAll(cmd, []byte{0}, []byte(" "))
	return string(cmd), nil
}

// EOF: "proc.go"
