package uuid

import (
	"os/exec"
	"strings"
)

func MakeUUID() (string, error) {
	uuid, err := exec.Command("uuidgen").Output()
	return strings.TrimSuffix(string(uuid), "\n"), err
}
