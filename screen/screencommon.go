package screen

import (
	"time"
)

type LinkStatusType uint8
type CommandType uint8

const LineWrapLen = 30
const MAX_SUBNETS = 64
const MAX_DNS_SERVERS = 2

const FullVPN = "FullVPN"
const SemiVPN = "SemiVPN"
const CustomVPN = "CustomVPN"

const (
	LINK_NOT_STARTED            LinkStatusType = 0
	LINK_DISCONNECTED           LinkStatusType = 1
	LINK_CONNECTION_IN_PROGRESS LinkStatusType = 2
	LINK_CONNECTED              LinkStatusType = 3
)

var LinkStatusMap = map[LinkStatusType]string{
	LINK_NOT_STARTED:            "Trying to connect. Please wait.",
	LINK_DISCONNECTED:           "Disconnected",
	LINK_CONNECTION_IN_PROGRESS: "Connection in progress. Please wait.",
	LINK_CONNECTED:              "Connected",
}

const (
	Connect             CommandType = 0
	Disconnect          CommandType = 1
	OnboardClient       CommandType = 2
	UploadLogs          CommandType = 3
	ResetConfig         CommandType = 4
	StopMain            CommandType = 5
	SaveConfig          CommandType = 6
	GetRouterIP         CommandType = 7
	GetWiFiClientIPList CommandType = 8
)

var CommandMap = map[CommandType]string{
	Connect:             "Connect",
	Disconnect:          "Disconnect",
	OnboardClient:       "OnboardClient",
	UploadLogs:          "UploadLogs",
	ResetConfig:         "ResetConfig",
	StopMain:            "StopMain",
	SaveConfig:          "SaveConfig",
	GetRouterIP:         "GetRouterIP",
	GetWiFiClientIPList: "GetWiFiClientIPList",
}

type NetworkEntry struct {
	VpnIp          uint32
	Relay          uint8
	Connected      bool
	In_bytes       uint64
	Out_bytes      uint64
	In_diff_bytes  uint64
	Out_diff_bytes uint64
	Name           string
}

type RouteEntry struct {
	Subnet string
	Via    string
	Mtu    string
	Name   string
}

type DNSEntry struct {
	server string
}

type Config struct {
	Vpnmode    string
	Rentries   []RouteEntry
	Servers    []DNSEntry
	DnsServers string
}

type IPListEntry struct {
	Name      string
	IPAddress string
}

type IPList struct {
	IPList []IPListEntry `json:iplist,omitempty`
}

type HomeDetails struct {
	Status      LinkStatusType
	Name        string
	Version     string
	CurVersion  string
	MyVPNIP     string
	RelayHostIP string
	LastUpdated time.Time
	MiscStatus  string
	Hosts       map[uint32]*NetworkEntry
}

type GUICommandCallback func(cmd CommandType, a []byte, length int) ([]byte, error)
