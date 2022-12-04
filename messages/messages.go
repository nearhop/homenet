package messages

import (
	"container/ring"
	"encoding/json"
	"fmt"

	nh_util "nh_util"

	"github.com/sirupsen/logrus"
)

const MAX_NUM_OF_CLIENTS = 128
const MAX_NUM_OF_BLOCK_CATEGORIES = 128

type m map[string]interface{}

// Connected client info
type ClientInfo struct {
	MACAddress    string `json:macaddress,omitempty`
	IPAddress     string `json:ipaddress,omitempty`
	Name          string `json:name,omitempty`
	Type          int    `json:type,omitempty`
	Channel       int    `json:channel,omitempty`
	SignalQuality int    `json:signalquality,omitempty`
	Paused        bool   `json:paused,omitempty`
	IsRepeater    bool   `json:isrepeater,omitempty`
	Lastseen      int64  `json:lastseen,omitempty`
}

type ClientsInfoMessage struct {
	Clients []ClientInfo `json:clients,omitempry`
}

type InnerMessage struct {
	Ssid2       string `json:ssid2,omitempty`
	Ssid5       string `json:ssid5,omitempty`
	Ssid52      string `json:ssid52,omitempty`
	Key2        string `json:key2,omitempty`
	Key5        string `json:key5,omitempty`
	Key52       string `json:key52,omitempty`
	Gssid2      string `json:gssid2,omitempty`
	Gssid5      string `json:gssid5,omitempty`
	Gssid52     string `json:gssid52,omitempty`
	Gkey2       string `json:gkey2,omitempty`
	Gkey5       string `json:gkey5,omitempty`
	Gkey52      string `json:gkey52,omitempty`
	Chan2       string `json:chan2,omitempty`
	Chan51      string `json:chan5,omitempty`
	Chan52      string `json:chan5,omitempty`
	Chan6       string `json:chan6,omitempty`
	Disabled2   string `json:disabled2,omitempty`
	Disabled5   string `json:disabled5,omitempty`
	Disabled52  string `json:disabled52,omitempty`
	Gdisabled2  string `json:gdisabled2,omitempty`
	Gdisabled5  string `json:gdisabled5,omitempty`
	Gdisabled52 string `json:gdisabled52,omitempty`
	Meshid      string `json:meshid,omitempty`
	Meshkey     string `json:meshkey,omitempty`
	Chanwidth2  string `json:chanwidth2,omitempty`
	Chanwidth51 string `json:chanwidth5,omitempty`
	Chanwidth52 string `json:chanwidth5,omitempty`
	Chanwidth6  string `json:chanwidth6,omitempty`
	Encryption  string `json:encryption,omitempty`
	Gencryption string `json:gencryption,omitempty`
	Fwupdate    string `json:fwupdate,omitempty`
	Fwversion   string `json:fwupdate,omitempty`
	Model       string `json:model,omitempty`
	Uptime      string `json:uptime,omitempty`
}
type ClientsInnerMessage struct {
	Clients []ClientInfo `json:clients,omitempty`
}

type BlockListMessageEntry struct {
	Domain  string `json:domain,omitempty`
	Blocked string `json:blocked,omitempty`
}

type BlocklistInnerMessage struct {
	Domains []BlockListMessageEntry `json:domains,omitempty`
}

type Message struct {
	Type  string       `json:type`
	Mbody InnerMessage `json:Mbody`
}

type InnerPauseMessage struct {
	MACAddress string `json:macaddress,omitempty`
	Pause      bool   `json:puase,omitempty`
}

type PauseMessage struct {
	Type  string            `json:type`
	Mbody InnerPauseMessage `json:Mbody`
}

type InnerClientMessage struct {
	MACAddress string `json:macaddress,omitempty`
	Name       string `json:name,omitempty`
	Type       uint8  `json:type,omitempty`
}

type ClientMessage struct {
	Type  string             `json:type`
	Mbody InnerClientMessage `json:Mbody`
}

type InnerStatsMessage struct {
	MACAddress string `json:macaddress,omitempty`
}

type StatsMessage struct {
	Type  string            `json:type`
	Mbody InnerStatsMessage `json:Mbody`
}

type ClientsMessage struct {
	Type  string              `json:type`
	Mbody ClientsInnerMessage `json:Mbody`
}

type BlocklistMessage struct {
	Type  string                `json:type`
	Mbody BlocklistInnerMessage `json:Mbody`
}

type InnerRouterEventMessage struct {
	Etype      int    `json:etype,omitempty`
	IPAddress  string `json:ipaddress,omitempty`
	MACAddress string `json:macaddress,omitempty`
	Name       string `json:name,omitempty`
	Extra      string `json:extra,omitempty`
	Tstamp     int64  `json:tstamp,omitempty`
}

type RouterEventMessage struct {
	Type  string                  `json:type,omitempty`
	Mbody InnerRouterEventMessage `json:mbody,omitempty`
}

type Event struct {
	EType       int
	EName       string
	EIPAddress  string
	EMACAddress string
	Extra       string
	Active      bool
	Tstamp      int64
}

func process_get_wireless_message() string {
	var message Message

	err := Get_wireless_message(&(message.Mbody))
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}

	jsonData, err := json.Marshal(message.Mbody)
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(jsonData)
}

func Get_client_message(mbody *ClientsInnerMessage) error {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/clients", []byte(""))
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &mbody.Clients)
	if err != nil {
		return err
	}
	return nil
}

func process_clients_get_message(l *logrus.Logger) string {
	var message ClientsMessage

	err := Get_client_message(&(message.Mbody))
	if err != nil {
		l.Error("Error while getting client message ", message.Mbody)
		return nh_util.NH_getErrorStatusString(err.Error())
	}

	jsonData, err := json.Marshal(message.Mbody)
	if err != nil {
		l.Error("Error while Marshalling client message ", message.Mbody)
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(jsonData)
}

func get_client_stats(req []byte) string {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/clistats", req)
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(data)
}

func get_repeaters() string {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/repeaters", []byte(""))
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(data)
}

func pause_client(req []byte) string {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/pause", req)
	if err != nil {
		fmt.Println("Error...", err.Error())
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(data)
}

func set_client_details(req []byte) string {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/setclientdetails", req)
	if err != nil {
		fmt.Println("Error...", err.Error())
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(data)
}

func upgrade_fw() string {
	args := []string{"&"}
	status := nh_util.NH_read_cmd_output(fw_upgrade_cmd, args)
	return "{\"status\": \"" + status + "\"}"
}

func upload_logs() string {
	data, err, _ := nh_util.Nh_http_send_req("http://127.0.0.1:11000/uploadlogs", []byte(""))
	if err != nil {
		fmt.Println("Error...", err.Error())
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	return string(data)
}

func newEvent(Events *ring.Ring, l *logrus.Logger, etype int, ip string, mac string, name string, extra string, tstamp int64) {
	event := &Event{
		EType:       etype,
		EIPAddress:  ip,
		EMACAddress: mac,
		EName:       name,
		Extra:       extra,
		Tstamp:      tstamp,
		Active:      true,
	}
	l.Error("Event...", event)
	Events.Value = event
	Events = Events.Next()
}

func handle_router_event(data []byte, l *logrus.Logger, Events *ring.Ring) string {
	var routerEventMessage RouterEventMessage
	err := json.Unmarshal(data, &routerEventMessage)
	if err != nil {
		return nh_util.NH_getErrorStatusString(err.Error())
	}
	newEvent(Events, l, routerEventMessage.Mbody.Etype, routerEventMessage.Mbody.IPAddress,
		routerEventMessage.Mbody.MACAddress, routerEventMessage.Mbody.Name,
		routerEventMessage.Mbody.Extra, routerEventMessage.Mbody.Tstamp)
	return ""
}

func ProcessMessage(data []byte, l *logrus.Logger, Events *ring.Ring) string {
	var message Message

	err := json.Unmarshal(data, &message)
	if err != nil {
		l.Error("Error while unmarshalling the received message", string(data))
		var pauseMessage PauseMessage
		err1 := json.Unmarshal(data, &pauseMessage)
		if err1 != nil {
			l.Error("Error while unmarshalling the received message again ", string(data))
			return nh_util.NH_getErrorStatusString(err.Error())
		}
	}

	switch message.Type {
	case "set_wireless":
		return Set_wireless(message.Mbody)
	case "get_wireless":
		return process_get_wireless_message()
	case "get_clients":
		return process_clients_get_message(l)
	case "set_client_details":
		return set_client_details(data)
	case "start_onboarding":
		return start_onboarding_ap(1)
	case "stop_onboarding":
		return start_onboarding_ap(0)
	case "pause_client":
		fallthrough
	case "pause_all":
		return pause_client(data)
	case "get_blocklist":
		return get_blocklist()
	case "set_blocklist":
		var blocklistMessage BlocklistMessage
		err = json.Unmarshal(data, &blocklistMessage)
		if err != nil {
			return nh_util.NH_getErrorStatusString(err.Error())
		}
		return set_blocklist(blocklistMessage.Mbody)
	case "get_client_stats":
		return get_client_stats(data)
	case "get_repeaters":
		return get_repeaters()
	case "upgrade_fw":
		return upgrade_fw()
	case "router_event":
		return handle_router_event(data, l, Events)
	case "upload_logs":
		return upload_logs()
	default:
		break
	}

	return ""
}
