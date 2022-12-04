package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	messages "messages"
	nh_util "nh_util"
)

type m map[string]interface{}

func main() {
	ipaddress := flag.String("ipaddress", "", "Destination IP Address")
	iname := flag.String("iname", "", "Bridge interface name of the repeater")
	miname := flag.String("miname", "", "Mesh interface name of the repeater")
	command := flag.String("command", "", "Command")
	fwver := flag.String("fwver", "", "Firmware version")
	data := flag.String("data", "", "Data")
	flag.Parse()

	if *ipaddress == "" {
		fmt.Errorf("IP Address is mandatory")
		flag.Usage()
		return
	}
	if *command == "" {
		fmt.Errorf("Command is mandatory")
		flag.Usage()
		return
	}
	var jc m

	switch *command {
	case "get_wireless":
		fallthrough
	case "check_wireless":
		jc = m{
			"type":  "get_wireless",
			"dummy": "dummy",
		}

		jsonData, err := json.Marshal(jc)
		if err != nil {
			fmt.Errorf("Error while marshalling data...", err.Error())
			return
		}

		message, err, _ := nh_util.Nh_http_send_req("http://"+*ipaddress+":11000/command", jsonData)
		if err == nil {
			if *command != "check_wireless" {
				fmt.Println(string(message))
			}
		} else {
			fmt.Println("Error: ", err.Error())
		}
		if *command == "check_wireless" {
			var msg messages.InnerMessage
			var rmsg messages.InnerMessage
			err = messages.Get_wireless_message(&msg)
			if err != nil {
				fmt.Println("Error while getting local wireless settings")
				return
			}
			err = json.Unmarshal(message, &rmsg)
			if err != nil {
				fmt.Println("Error while unmarhsalling the wireless messages")
				return
			}
			if rmsg.Ssid2 == "" || rmsg.Ssid5 == "" ||
				rmsg.Key2 == "" || rmsg.Key5 == "" ||
				rmsg.Gssid2 == "" || rmsg.Gssid5 == "" ||
				rmsg.Gkey2 == "" || rmsg.Gkey5 == "" ||
				rmsg.Meshid == "" || rmsg.Meshkey == "" ||
				rmsg.Encryption == "" || rmsg.Gencryption == "" {
				fmt.Println("Error while getting wireless messages from root. Got blank values")
				return
			}
			if msg.Ssid2 == rmsg.Ssid2 && msg.Ssid5 == rmsg.Ssid5 && msg.Ssid52 == rmsg.Ssid52 &&
				msg.Key2 == rmsg.Key2 && msg.Key5 == rmsg.Key5 && msg.Key52 == rmsg.Key52 &&
				msg.Gssid2 == rmsg.Gssid2 && msg.Gssid5 == rmsg.Gssid5 && msg.Gssid52 == rmsg.Gssid52 &&
				msg.Gkey2 == rmsg.Gkey2 && msg.Gkey5 == rmsg.Gkey5 && msg.Gkey52 == rmsg.Gkey52 &&
				msg.Chanwidth2 == rmsg.Chanwidth2 && msg.Chanwidth51 == rmsg.Chanwidth51 && msg.Chanwidth52 == rmsg.Chanwidth52 &&
				msg.Meshid == rmsg.Meshid && msg.Meshkey == rmsg.Meshkey &&
				msg.Encryption == rmsg.Encryption && msg.Gencryption == rmsg.Gencryption &&
				msg.Disabled2 == rmsg.Disabled2 && msg.Disabled5 == rmsg.Disabled5 && msg.Disabled52 == rmsg.Disabled52 {
				if rmsg.Fwupdate == "1" {
					fmt.Println("UpdateFirmware")
					return
				}
				return
			}
			fmt.Println("Changed")
		}
		return
	case "set_wireless":
		var msg messages.InnerMessage
		if *data == "" {
			fmt.Errorf("Wireless set needs filename containing the wireless config")
			return
		}

		config, err := nh_util.NH_read_file(*data)
		if err != nil {
			fmt.Println("Error Wireless config file does n't exist or some error", err.Error())
			return
		}
		err = json.Unmarshal(config, &msg)
		if err != nil {
			fmt.Println("Error while unmarhsalling the wireless messages")
			return
		}
		messages.Set_wireless(msg)
		return
	case "register_repeater":
		mac, err := nh_util.NH_get_macaddress(*iname)
		if err != nil {
			fmt.Println("Not able to get Interface MAC Address")
			return
		}
		mmac, err := nh_util.NH_get_macaddress(*miname)
		if err != nil {
			fmt.Println("Not able to get Mesh Interface MAC Address")
			return
		}
		ip, err := nh_util.NH_get_ipv4address(*iname)
		if err != nil {
			fmt.Println("Not able to get Interface IPv4 Address")
			return
		}
		hname, err := os.Hostname()
		if err != nil {
			fmt.Println("Error while getting hostname")
			return
		}
		jc = m{
			"type":  "register_repeater",
			"name":  hname,
			"mac":   mac,
			"mmac":  mmac,
			"ip":    ip,
			"fwver": fwver,
		}

		jsonData, err := json.Marshal(jc)
		if err != nil {
			fmt.Println("Error while marshalling data...", err.Error())
			return
		}

		message, err, _ := nh_util.Nh_http_send_req("http://"+*ipaddress+":11000/register_repeater", jsonData)
		if err != nil {
			fmt.Println("Error while sending register_repeater to the server")
			return
		}
		fmt.Println(string(message))
	default:
		fmt.Errorf("Command Not supported yet")
		return
	}
}
