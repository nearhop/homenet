//go:build router
// +build router

package messages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	nh_util "nh_util"
)

func get_wireless() string {
	out, err := exec.Command(get_wireless_script).Output()

	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	out1 := strings.TrimSuffix(string(out), "\n")
	return string(out1)
}

func parseSettings(settings string) (map[string]string, error) {
	entries := strings.Split(settings, "\n")
	settingsMap := make(map[string]string)
	if settingsMap == nil {
		return nil, fmt.Errorf("Error while allocating settings map")
	}
	for _, entry := range entries {
		pair := strings.Split(entry, "=")
		var key string
		var value string

		if len(pair) == 0 {
			continue
		}
		key = pair[0]
		if len(pair) < 2 {
			value = ""
		} else {
			value = pair[1]
		}
		settingsMap[key] = value
	}
	return settingsMap, nil
}

func Get_wireless_message(message *InnerMessage) error {
	settings := get_wireless()
	if settings == "" {
		return fmt.Errorf("Error while getting wireless settings")
	}

	settingsMap, err := parseSettings(settings)
	if err != nil {
		return fmt.Errorf("Error while parsing wireless settings")
	}
	message.Ssid2 = settingsMap["ssid2"]
	message.Ssid5 = settingsMap["ssid5"]
	message.Ssid52 = settingsMap["ssid52"]
	message.Key2 = settingsMap["key2"]
	message.Key5 = settingsMap["key5"]
	message.Key52 = settingsMap["key52"]
	message.Gssid2 = settingsMap["gssid2"]
	message.Gssid5 = settingsMap["gssid5"]
	message.Gssid52 = settingsMap["gssid52"]
	message.Gkey2 = settingsMap["gkey2"]
	message.Gkey5 = settingsMap["gkey5"]
	message.Gkey52 = settingsMap["gkey52"]
	message.Disabled2 = settingsMap["disabled2"]
	message.Disabled5 = settingsMap["disabled5"]
	message.Disabled52 = settingsMap["disabled52"]
	message.Gdisabled2 = settingsMap["gdisabled2"]
	message.Gdisabled5 = settingsMap["gdisabled5"]
	message.Gdisabled52 = settingsMap["gdisabled52"]
	message.Meshid = settingsMap["meshid"]
	message.Meshkey = settingsMap["meshkey"]
	message.Chan2 = settingsMap["chan2"]
	message.Chan51 = settingsMap["chan51"]
	message.Chan52 = settingsMap["chan52"]
	message.Chan6 = settingsMap["chan6"]
	message.Chanwidth2 = settingsMap["chanwidth2"]
	message.Chanwidth51 = settingsMap["chanwidth51"]
	message.Chanwidth52 = settingsMap["chanwidth52"]
	message.Chanwidth6 = settingsMap["chanwidth6"]
	message.Encryption = settingsMap["encryption"]
	message.Gencryption = settingsMap["gencryption"]
	message.Fwupdate = settingsMap["fwupdate"]
	message.Fwversion = settingsMap["fwversion"]
	message.Model = settingsMap["model"]
	message.Uptime = settingsMap["uptime"]

	return nil
}

func Set_wireless(message InnerMessage) string {
	exec.Command(set_wireless_script,
		"--ssid2", message.Ssid2,
		"--ssid5", message.Ssid5,
		"--ssid52", message.Ssid52,
		"--key2", message.Key2,
		"--key5", message.Key5,
		"--key52", message.Key52,
		"--gssid2", message.Gssid2,
		"--gssid5", message.Gssid5,
		"--gssid52", message.Gssid52,
		"--gkey2", message.Gkey2,
		"--gkey5", message.Gkey5,
		"--gkey52", message.Gkey52,
		"--chan2", message.Chan2,
		"--chan51", message.Chan51,
		"--chan52", message.Chan52,
		"--chan6", message.Chan6,
		"--chanwidth2", message.Chanwidth2,
		"--chanwidth51", message.Chanwidth51,
		"--chanwidth52", message.Chanwidth52,
		"--chanwidth6", message.Chanwidth6,
		"--chanwidth6", message.Chanwidth6,
		"--encryption", message.Encryption,
		"--gencryption", message.Gencryption,
		"--disabled2", message.Disabled2,
		"--disabled5", message.Disabled5,
		"--disabled52", message.Disabled52,
		"--meshssid", message.Meshid,
		"--meshkey", message.Meshkey,
	).Run()

	status := "{\"status\": \"success\"}"
	return status
}

func get_blocklist() string {
	var Domains [MAX_NUM_OF_BLOCK_CATEGORIES]BlockListMessageEntry

	args := []string{}
	cmd := get_blocklist_cmd
	settings := nh_util.NH_read_cmd_output(cmd, args)
	if settings == "" {
		status := "Error while getting blocklist settings"
		return nh_util.NH_getErrorStatusString(status)
	}

	settingsMap, err := parseSettings(settings)
	if err != nil {
		return "Error while parsing blocklist settings"
	}
	Domains[0].Domain = "malware"
	Domains[0].Blocked = settingsMap["blockmalware"]
	Domains[1].Domain = "adult"
	Domains[1].Blocked = settingsMap["blockadult"]
	Domains[2].Domain = "amongus"
	Domains[2].Blocked = settingsMap["blockamongus"]
	Domains[3].Domain = "banuba"
	Domains[3].Blocked = settingsMap["blockbanuba"]
	Domains[4].Domain = "facebook"
	Domains[4].Blocked = settingsMap["blockfacebook"]
	Domains[5].Domain = "instagram"
	Domains[5].Blocked = settingsMap["blockinstagram"]
	Domains[6].Domain = "parlor"
	Domains[6].Blocked = settingsMap["blockparlor"]
	Domains[7].Domain = "roblox"
	Domains[7].Blocked = settingsMap["blockroblox"]
	Domains[8].Domain = "snapchat"
	Domains[8].Blocked = settingsMap["blocksnapchat"]
	Domains[9].Domain = "tellonym"
	Domains[9].Blocked = settingsMap["blocktellonym"]
	Domains[10].Domain = "tiktok"
	Domains[10].Blocked = settingsMap["blocktiktok"]
	Domains[11].Domain = "tinder"
	Domains[11].Blocked = settingsMap["blocktinder"]
	Domains[12].Domain = "youtube"
	Domains[12].Blocked = settingsMap["blockyoutube"]
	Domains[13].Domain = "zoomerang"
	Domains[13].Blocked = settingsMap["blockzoomerang"]
	Domains[14].Domain = "discord"
	Domains[14].Blocked = settingsMap["blockdiscord"]
	Domains[15].Domain = "fifamobile"
	Domains[15].Blocked = settingsMap["blockfifamobile"]

	jsonData, err := json.Marshal(Domains[:16])
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(jsonData)
}

func set_blocklist(message BlocklistInnerMessage) string {
	args := make([]string, 2*len(message.Domains))
	if args == nil {
		return "set_blocklist: Error while allocating memeory for args"
	}
	for i := 0; i < len(message.Domains); i++ {
		args[2*i] = message.Domains[i].Domain
		args[2*i+1] = message.Domains[i].Blocked
	}
	exec.Command(set_blocklist_cmd, args...).Run()
	return ""
}
