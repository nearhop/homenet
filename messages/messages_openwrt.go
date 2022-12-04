//go:build openwrt
// +build openwrt

package messages

import (
	"strconv"

	nh_util "nh_util"
)

const set_wireless_script = "/sbin/set_wireless.sh"
const get_wireless_script = "/sbin/get_wireless.sh"
const get_blocklist_cmd = "/sbin/get_block_urllist.sh"
const set_blocklist_cmd = "/sbin/set_block_urllist.sh"
const fw_upgrade_cmd = "/sbin/fw_update.sh"

func openwrt_process_wireless_message(json_message string) string {
	return ""
}

func start_onboarding_ap(start int) string {
	args1 := strconv.Itoa(start)
	args := []string{args1}
	cmd := "/sbin/run_on_board_ap.sh"
	nh_util.NH_read_cmd_output(cmd, args)

	status := "{\"status\": \"success\"}"
	return status
}
