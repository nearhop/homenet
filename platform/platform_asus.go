//go:build asus
// +build asus

package platform

import (
	"strings"

	nh_util "nh_util"
)

const (
	Resolvfile = "/tmp/dummy"
)

func Get_deviceid() string {
	args := []string{"/sys/class/net/eth0/address"}
	cmd := "cat"
	out := nh_util.NH_read_cmd_output(cmd, args)

	out1 := strings.TrimSuffix(string(out), "\n")
	return string(out1)
}

func Get_osname() string {
	return "asuswrt"
}

func Get_capture_device() string {
	return "br0"
}

func GetDefaultInterfaceName() (str string, err error) {
	return "", nil
}

func Get_Device_Type() string {
	return "20"
}
