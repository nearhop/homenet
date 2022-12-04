//go:build openwrt
// +build openwrt

package platform

import (
	"os/exec"
	"strings"
)

const (
	Resolvfile = "/tmp/dummy"
)

func Get_deviceid() string {
	out, err := exec.Command("/sbin/get_machineid.sh").Output()

	if err != nil {
		return "na"
	}
	out1 := strings.TrimSuffix(string(out), "\n")
	return string(out1)
}

func Get_osname() string {
	return "openwrt"
}

func Get_capture_device() string {
	return "br-lan"
}

func GetDefaultInterfaceName() (str string, err error) {
	return "", nil
}

func Get_Device_Type() string {
	return "20"
}
