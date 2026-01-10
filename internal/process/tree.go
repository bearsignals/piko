package process

import (
	"os/exec"
	"strconv"
	"strings"
)

func GetParentPID(pid int) (int, error) {
	out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

func FindTmuxPane(pid int) (string, error) {
	panes, err := listTmuxPanes()
	if err != nil {
		return "", err
	}

	currentPID := pid
	for i := 0; i < 10; i++ {
		if paneID, ok := panes[currentPID]; ok {
			return paneID, nil
		}

		parentPID, err := GetParentPID(currentPID)
		if err != nil || parentPID <= 1 {
			break
		}
		currentPID = parentPID
	}

	return "", nil
}

func listTmuxPanes() (map[int]string, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_pid} #{pane_id}").Output()
	if err != nil {
		return nil, err
	}

	panes := make(map[int]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		panes[pid] = parts[1]
	}

	return panes, nil
}
