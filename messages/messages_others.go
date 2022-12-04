//go:build !openwrt && !asus
// +build !openwrt,!asus

package messages

const fw_upgrade_cmd = ""

func Get_wireless_message(message *InnerMessage) error {
	return nil
}

func Set_wireless(message InnerMessage) string {
	return ""
}

func start_onboarding_ap(start int) string {
	return ""
}

func get_blocklist() string {
	return ""
}

func set_blocklist(message BlocklistInnerMessage) string {
	return ""
}
