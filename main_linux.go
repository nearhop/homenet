//go:build linux && !asus
// +build linux,!asus

package nebula

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func getProcessOwner() string {
	stdout, err := exec.Command("ps", "-o", "user=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return "na"
	}
	return strings.TrimSuffix(string(string(stdout)), "\n")
}

func IsRootUser() bool {
	if getProcessOwner() == "root" {
		return true
	} else {
		return false
	}
}

func GetConfigFileDir() string {
	return "/etc/nearhop/configs/"
}

func GetLogsFileDir() string {
	return "/etc/nearhop/logs/"
}

func GetNearhopDir() string {
	return "/etc/nearhop/"
}

func GetModel() string {
	return "linux"
}
