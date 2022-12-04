//go:build asus
// +build asus

package router

import (
	"strings"
	"time"

	nh_util "nh_util"
)

const DB_CLIENTS_LOCATION = "/jffs/nearhop/clients/"
const DB_REPEATERS_FILE = "/jffs/nearhop/repeaters.json"
const get_hostname_cmd = "/jffs/nearhop/sbin/get_hostname.sh"
const nearhop_hostnames = "/jffs/nearhop/router_configs/hostnames.txt"
const updateBlockedURLsScript = "/jffs/nearhop/sbin/update_blocked_urls.sh"

func Router_onboarded(opmode string) {
	// Wait for 5 seconds. By this time the app should get the response
	// Hence even if the app is disconnected because of wifi/network restart,
	// it would get the http response (calling function of this function dumps it)
	time.Sleep(5 * time.Second)
	args := []string{opmode}
	cmd := "/sbin/onboarded.sh"
	nh_util.NH_read_cmd_output(cmd, args)
}

func getFileName(mac string) string {
	filename := DB_CLIENTS_LOCATION + strings.Replace(mac, ":", "_", -1)
	return filename
}

func pauseClient(mac string, ip string, name string, pause bool) string {
	var pauseString string
	if pause {
		pauseString = "1"
	} else {
		pauseString = "0"
	}
	args := []string{mac, ip, pauseString, name}
	cmd := "/jffs/nearhop/sbin/pause_client.sh"
	return nh_util.NH_read_cmd_output(cmd, args)
}

func pauseAll(pause bool) string {
	var pauseString string
	if pause {
		pauseString = "1"
	} else {
		pauseString = "0"
	}
	args := []string{pauseString}
	cmd := "/sbin/pause_all.sh"
	return nh_util.NH_read_cmd_output(cmd, args)
}

func getSignalQuality(rssi int) int {
	if rssi > -40 {
		return SIGNAL_QUALITY_EXCELLENT
	} else if rssi > -75 {
		return SIGNAL_QUALITY_GOOD
	} else {
		return SIGNAL_QUALITY_AVERAGE
	}
}

func addDNSEntryPlatform(client *RouterClient) {
}

func applyDNSEntries() {
	args := []string{""}
	cmd := "/jffs/nearhop/sbin/apply_dns.sh"
	nh_util.NH_read_cmd_output(cmd, args)
}
