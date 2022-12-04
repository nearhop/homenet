//go:build router
// +build router

package router

type ClientType uint8
type EventType uint8

const (
	WIRED          ClientType = 0
	WIRELESS       ClientType = 1
	WIRELESSLAPTOP ClientType = 2
	WIRELESSPHONE  ClientType = 3
	WIRELESSIOT    ClientType = 4
)

const (
	NEWCLIENTEVENT EventType = 0
	BLOCKEDIPEVENT EventType = 1
)

const MAX_CLIENT_MINUTE_STATS_ENTRIES = 30
const MAX_CLIENT_HOUR_STATS_ENTRIES = 4
const TIMESTAMP_FORMAT = "2006-02-01 15:04"
const TIMESTAMP_FORMAT_HOUR = "2006-02-01 15"
const MAX_NUM_OF_REPEATERS = 5
const MAX_NUMBER_OF_EVENTS = 8

const (
	SIGNAL_QUALITY_EXCELLENT int = 0
	SIGNAL_QUALITY_GOOD      int = 1
	SIGNAL_QUALITY_AVERAGE   int = 2
)

type Interface struct {
	Ifname    string `json:ifname`
	Mac       string `json:mac`
	Ip        string `json:ip`
	Mask      string `json:mask`
	Wan       int    `json:wan`
	Kind      int    `json:kind`
	Slice     int    `json:slice`
	End_sec   uint32 `json:end_sec`
	End_msec  uint32 `json:end_msec`
	Len_msec  uint32 `json:len_msec`
	Bytes_in  uint32 `json:bytes_in`
	Bytes_out uint32 `json:bytes_out`
	Proc_pkts uint32 `json:proc_pkts`
	Pcap_pkts uint32 `json:pcap_pkts`
	Miss_pkts uint32 `json:miss_pkts`
}

type RepeaterMessage struct {
	Type  string `json:type`
	Name  string `json:Name`
	Mac   string `json:mac`
	MMac  string `json:mmac`
	Ip    string `json:ip`
	Fwver string `json:fwver`
}

type Con struct {
	R_port int    `json:r_port`
	R_ip   string `json:r_ip`
	Proto  int    `json:proto`
	B_in   []int  `json:b_in`
	B_out  []int  `json:b_out`
}

type Device struct {
	Ifname string `json:ifname`
	Ip     string `json:ip`
	Mac    string `json:mac`
	Conn   []Con  `json:conn`
}

type TelemetryData struct {
	Devices    []Device    `json:devices`
	Interfaces []Interface `json:interfaces`
}

type Station struct {
	Mac  string `json:mac`
	Rssi int    `json:rssi`
	//	Extras string `json:extras`
}

type VAP struct {
	Interface string `json:interface`
	Ssid      string `json:ssid`
	Bssid     string `json:bssid`
	Mode      string `json:mode`
	Kind      int    `json:kind`
	//	Extras    string    `jsong:extras`
	Stas []Station `json:stas`
}

type Radio struct {
	Name    string `jsong:name`
	Channel int    `jsong:channel`
	//	Extras  string `jsong:extras`
	Vaps []VAP `jsong:vaps`
}

type WirelessTelemetryData struct {
	Radios []Radio `json:radios`
}

// Connected client info
type RouterClient struct {
	MACAddress    string
	IPAddress     string
	Name          string
	Fwver         string
	IsRepeater    bool
	Type          ClientType
	Channel       int
	Rssi          int
	Dirty         bool
	Paused        bool
	name_attempts int
	Lastseen      int64
}
type addClient func(client *RouterClient)

type Repeater struct {
	Name  string `json:name`
	Mac   string `json:mac`
	MMac  string `json:mmac`
	Ip    string `json:ip`
	Fwver string `json:fwver`
}

type RouterEvent struct {
	Active bool
	Etype  EventType
	Extra  string
	Client *RouterClient
	Tstamp int64
}
