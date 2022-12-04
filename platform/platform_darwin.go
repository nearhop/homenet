//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

func Get_osname() string {
	return "mac"
}

func parseDarwinRouteGet(output []byte) (string, error) {
	// Darwin route out format is always like this:
	//    route to: default
	// destination: default
	//        mask: default
	//     gateway: 192.168.1.1
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "interface:" {
			return fields[1], nil
		}
	}

	return "", fmt.Errorf("No interface")
}

func GetDefaultInterfaceName() (str string, err error) {
	routeCmd := exec.Command("/sbin/route", "-n", "get", "0.0.0.0")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return parseDarwinRouteGet(output)
}
