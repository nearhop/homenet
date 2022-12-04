//go:build asus
// +build asus

package messages

const set_wireless_script = "/jffs/nearhop/sbin/set_wireless.sh"
const get_wireless_script = "/jffs/nearhop/sbin/get_wireless.sh"
const get_blocklist_cmd = "/jffs/nearhop/sbin/get_block_urllist.sh"
const set_blocklist_cmd = "/jffs/nearhop/sbin/set_block_urllist.sh"
const fw_upgrade_cmd = "/jffs/nearhop/sbin/fw_update.sh"

func start_onboarding_ap(start int) string {
	return ""
}
