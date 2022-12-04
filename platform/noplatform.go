//go:build !openwrt && !asus
// +build !openwrt,!asus

package platform

import (
	"os/exec"
	"strings"
)

func Get_deviceid() string {
	out, err := exec.Command("/sbin/get_machineid.sh").Output()

	if err != nil {
		return "na"
	}
	out1 := strings.TrimSuffix(string(out), "\n")
	return string(out1)
}

func Get_capture_device() string {
	return ""
}

func Get_Device_Type() string {
	return "10"
}
