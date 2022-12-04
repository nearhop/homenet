package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	messages "messages"
	nh_util "nh_util"
	router "router"
	screen "screen"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/util"
	"gopkg.in/yaml.v2"
)

const routeripsurl = nh_util.Homeneturl + "hncapi/getrouterip"
const mobileipsurl = nh_util.Homeneturl + "hncapi/getmobileips"
const versionurl = nh_util.Homeneturl + "hncapi/cldevstatus"

type mp map[string]interface{}

type MainActivity struct {
	ctrl        *nebula.Control
	l           *logrus.Logger
	Version     string
	CurVersion  string
	config      *config.C
	mw          *screen.MainWindow
	trigger     chan bool
	Status      screen.LinkStatusType
	Name        string
	MyVPNIP     string
	RelayHostIP string
	LastUpdated time.Time
	onboarded   bool
	configPath  string
	publicKey   string
	privateKey  string
	ca          string
	Servers     []nebula.ServerEntry
	deviceip    string
	status_err  string
	hosts       map[uint32]*screen.NetworkEntry
	prevhosts   map[uint32]*screen.NetworkEntry
	rs          *router.RouterServer
	MobileIPs   []string
	routerip    string
}

type OnboardData struct {
	Email string `json:email`
	Key   string `json:key`
	Name  string `json:name`
}

type ConfigData struct {
	Configsave screen.Config `json:configsave`
}

type RouterIP struct {
	Id            string `json:_id`
	DeviceName    string `json:deviceName`
	DeviceIp      string `json:deviceIp`
	PrimaryRouter int    `json:primaryRouter`
}

type RouterIPReply struct {
	Status  int        `json:status`
	Message []RouterIP `json:message`
}

type MobileIP struct {
	Id         string `json:_id`
	DeviceId   string `json:deviceId`
	ClDeviceId string `json:clDeviceId`
	DeviceName string `json:deviceName`
	DeviceIp   string `json:deviceIp`
	Nwid       string `json:nwid`
}

type MobileIPReply struct {
	Status  int        `json:status`
	Message []MobileIP `json:message`
}

type DeviceStatus struct {
	Model       string `json:model`
	V           string `json:__v`
	ForceUpdate bool   `json:forceUpdate`
	FwSignedUrl string `json:fwSignedUrl`
	Fwver       string `json:fwver`
	updatedAt   string `json:updatedAt`
}

type DevStatusReply struct {
	Status  int          `json:status`
	Message DeviceStatus `json:message`
}

type m map[string]interface{}

func NewMainActivity(logs *logrus.Logger, Build string, c *config.C, m *screen.MainWindow, ob bool, configfile string) *MainActivity {
	return &MainActivity{
		trigger:    make(chan bool),
		Version:    Build,
		CurVersion: "",
		l:          logs,
		config:     c,
		mw:         m,
		Name:       "",
		onboarded:  ob,
		ctrl:       nil,
		configPath: configfile,
		hosts:      map[uint32]*screen.NetworkEntry{},
		prevhosts:  map[uint32]*screen.NetworkEntry{},
	}
}

func (m *MainActivity) RenderConfig(certs *nebula.Certs) (string, error) {
	config := newConfig()

	config.PKI.CA = m.ca
	config.PKI.Cert = m.publicKey
	config.PKI.Key = m.privateKey
	config.PKI.Token = certs.Token
	config.PKI.DeviceId = certs.DeviceId
	config.Listen.Port = 4242
	config.Lighthouse.Interval = 60
	mtu := 1300
	config.Tun.MTU = &mtu

	for _, server := range m.Servers {
		hosts2 := make([]string, 1)
		if hosts2 == nil {
			m.l.Error("Error while rendering config. Can't allocate space for Relay servers")
			return "", fmt.Errorf("Error while rendering config. Can't allocate space for Relay servers")
		}
		hosts2[0] = server.ServerIP + ":" + server.Port
		config.StaticHostmap[server.ServerPvtIP] = hosts2
	}
	for _, server := range m.Servers {
		config.Lighthouse.Hosts = append(config.Lighthouse.Hosts, server.ServerPvtIP)
	}
	finalConfig, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(finalConfig), nil
}

func (m *MainActivity) getCert() string {
	pub, priv := nebula.X25519Keypair()

	m.publicKey = string(cert.MarshalX25519PublicKey(pub))
	m.privateKey = string(cert.MarshalX25519PrivateKey(priv))

	return m.publicKey
}

func (m *MainActivity) generate_config(certs *nebula.Certs) error {
	m.ca = certs.Ca
	m.publicKey = certs.Cert
	m.Servers = certs.Servers
	m.deviceip = certs.DeviceIp
	config_string, err := m.RenderConfig(certs)
	if err != nil {
		return fmt.Errorf("Error while generating config..", err)
	}
	return m.save_config_to_file(config_string, nebula.GetConfigFileName())
}

func (m *MainActivity) save_config_to_file(config_string string, configPath string) error {
	config_file_dir := nebula.GetConfigFileDir()
	m.l.Info("Creating config file dir...", config_file_dir)
	_, err := os.Stat(config_file_dir)
	if err != nil {
		err = os.Mkdir(config_file_dir, 0755)
		if err != nil {
			return fmt.Errorf("Error while creating config directory")
		}
	}
	_, err = os.Stat(nebula.GetLogsFileDir())
	m.l.Info("Creating logs file dir...", nebula.GetLogsFileDir)
	if err != nil {
		err = os.Mkdir(nebula.GetConfigFileDir(), 0755)
		if err != nil {
			return fmt.Errorf("Error while creating config directory")
		}
	}
	err = ioutil.WriteFile(nebula.GetConfigFileDir()+configPath, []byte(config_string), 0644)
	if err != nil {
		return fmt.Errorf("Error while creating config ", config_string)
	}
	return nil
}

func (m *MainActivity) uploadLogs() error {
	logfile := nebula.GetLogsFileDir() + "logs.txt"
	filename, err := os.Readlink(logfile)
	if err != nil {
		return fmt.Errorf("Error while getting logs filename", err)
	}
	logs, err := ioutil.ReadFile(nebula.GetLogsFileDir() + filename)
	if err != nil {
		return fmt.Errorf("Error while uploading logs file", err)
	}
	networkID, err := nebula.GetNetworkID(m.config)
	if err != nil {
		m.l.Error(err.Error() + " uploadLogs: Error while getting networkid")
		return err
	}
	token := m.config.GetString("pki.token", "")
	if token == "" {
		m.l.Error("getIPList: Error while getting pki token")
		return fmt.Errorf("Error while getting pki token")
	}
	deviceid := m.config.GetString("pki.deviceid", "")
	if deviceid == "" {
		m.l.Error("getIPList: Error while getting deviceid")
		return fmt.Errorf("Error while getting pki token")
	}
	nwidstr := strconv.FormatUint(networkID, 10)
	jc := mp{
		"nwid":        nwidstr,
		"deviceId":    deviceid,
		"clienttoken": token,
		"devlog":      string(logs),
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		return err
	}
	_, err, _ = nh_util.Nh_http_send_req(nh_util.Homeneturl+"hncapi/devlog", jsonData)
	m.l.Error("Uploaded logs")
	return err
}

func (m *MainActivity) resetConfig() error {

	os.RemoveAll(nebula.GetLogsFileDir())
	os.RemoveAll(nebula.GetConfigFileDir())
	os.RemoveAll(nebula.GetNearhopDir())

	m.setStatus("Config has been reset")
	m.stopMain()
	return nil
}

func (m *MainActivity) stopMain() error {
	m.stop()
	os.Exit(0)

	return nil
}

func (m *MainActivity) saveConfig(config ConfigData) error {
	var unsafe_routes []configUnsafeRoute

	unsafe_routes = []configUnsafeRoute{}
	if config.Configsave.Vpnmode == screen.CustomVPN && len(config.Configsave.Rentries) > 0 {
		unsafe_routes = make([]configUnsafeRoute, len(config.Configsave.Rentries))
		if unsafe_routes == nil {
			return fmt.Errorf("Error while saving the config. Can't allocate memory")
		}
		index := 0
		for i := 0; i < len(config.Configsave.Rentries); i++ {
			exists := false
			for j := 0; j < index; j++ {
				if unsafe_routes[j].Route == config.Configsave.Rentries[i].Subnet {
					// Duplicate entry. Dont save
					exists = true
					break
				}
			}
			if !exists {
				unsafe_routes[index].Route = config.Configsave.Rentries[i].Subnet
				unsafe_routes[index].Via = config.Configsave.Rentries[i].Via
				unsafe_routes[index].Name = config.Configsave.Rentries[i].Name
				mtu, err := strconv.Atoi(config.Configsave.Rentries[i].Mtu)
				if err != nil {
					return fmt.Errorf("Error while saving config. Invalid MTU value", config.Configsave.Rentries[i].Mtu)
				}
				unsafe_routes[index].MTU = &mtu
				index++
			}
		}
		unsafe_routes = unsafe_routes[0:index]
	} else if config.Configsave.Vpnmode == screen.FullVPN {
		unsafe_routes = make([]configUnsafeRoute, 1)
		if unsafe_routes == nil {
			return fmt.Errorf("Error while saving the config. Can't allocate memory")
		}
		unsafe_routes[0].Route = "0.0.0.0/0"
		unsafe_routes[0].Via = config.Configsave.Rentries[0].Via // ToDo: Find a way to get the IP Address of the Router and remove this hardcoded value
		mtu := 1300
		unsafe_routes[0].MTU = &mtu
	}
	dirty_config := newConfig()
	if dirty_config == nil {
		return fmt.Errorf("Error while saving the config. Can't allocate memory for config")
	}
	if config.Configsave.Vpnmode == screen.CustomVPN {
		dirty_config.Tun.UnsafeRoutes = unsafe_routes
		dirty_config.Tun.Fullvpn = false
	} else if config.Configsave.Vpnmode == screen.FullVPN {
		dirty_config.Tun.UnsafeRoutes = unsafe_routes
		dirty_config.Tun.Fullvpn = true
	} else {
		dirty_config.Tun.Fullvpn = false
	}

	dirty_config.Tun.Dns = config.Configsave.DnsServers

	finalConfig, err := yaml.Marshal(dirty_config)
	if err != nil {
		return err
	}
	return m.save_config_to_file(string(finalConfig), nebula.GetConfigFileNameSecond())
}

func (m *MainActivity) getIPList(url string) ([]byte, error) {
	networkID, err := nebula.GetNetworkID(m.config)
	if err != nil {
		m.l.Error(err.Error() + " getIPList: Error while getting networkid")
		return nil, err
	}
	token := m.config.GetString("pki.token", "")
	if token == "" {
		m.l.Error("getIPList: Error while getting pki token")
		return nil, fmt.Errorf("Error while getting pki token")
	}
	deviceid := m.config.GetString("pki.deviceid", "")
	if deviceid == "" {
		m.l.Error("getIPList: Error while getting deviceid")
		return nil, fmt.Errorf("Error while getting pki token")
	}
	nwidstr := strconv.FormatUint(networkID, 10)
	jc := mp{
		"nwid":        nwidstr,
		"clienttoken": token,
		"deviceId":    deviceid,
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		m.l.Error(err.Error() + " getIPList: Error while marhsalling data")
		return nil, err
	}
	bytes, err, _ := nh_util.Nh_http_send_req(url, jsonData)
	if err != nil {
		m.l.Error(err.Error() + " getIPList: Error while receiving the data from the server")
		return nil, err
	}
	return bytes, nil
}

func (m *MainActivity) getDeviceStatus() ([]byte, error) {
	var reply DevStatusReply

	networkID, err := nebula.GetNetworkID(m.config)
	if err != nil {
		m.l.Error(err.Error() + " getDeviceStatus: Error while getting networkid")
		return nil, err
	}
	token := m.config.GetString("pki.token", "")
	if token == "" {
		m.l.Error("getDeviceStatus: Error while getting pki token")
		return nil, fmt.Errorf("Error while getting pki token")
	}
	deviceid := m.config.GetString("pki.deviceid", "")
	if deviceid == "" {
		m.l.Error("getDeviceStatus: Error while getting deviceid")
		return nil, fmt.Errorf("Error while getting pki token")
	}
	nwidstr := strconv.FormatUint(networkID, 10)
	jc := mp{
		"nwid":        nwidstr,
		"clienttoken": token,
		"deviceId":    deviceid,
		"deviceOS":    nebula.GetModel(),
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		m.l.Error(err.Error() + " getDeviceStatus: Error while marhsalling data")
		return nil, err
	}
	bytes, err, code := nh_util.Nh_http_send_req(versionurl, jsonData)
	if code == http.StatusUnauthorized {
		// This device is no more active. Reset the config
		m.resetConfig()
		return nil, err
	}
	if err != nil {
		m.l.Error(err.Error() + " getDeviceStatus: Error while receiving the data from the server")
		return nil, err
	}
	err = json.Unmarshal(bytes, &reply)
	if err != nil {
		m.l.Error(err.Error() + " getMobileIPs: Error while unmarhsalling data")
		return nil, err
	}

	if reply.Status != 1 {
		m.l.Error(" getDeviceStatus: Not a successful fetch")
		return nil, fmt.Errorf("getDeviceStatus: Not a successful fetch")
	}
	m.CurVersion = reply.Message.Fwver
	return bytes, nil
}

func (m *MainActivity) SendNonTunMessage(vpnIp iputil.VpnIp, jsonData []byte) ([]byte, error) {
	// 5 tries with 5 seconds in between each try
	var ret string
	var err error

	for counter := 0; counter < 5; counter++ {
		ret, err = m.ctrl.SendNonTunMessage(vpnIp, jsonData)
		if err == nil {
			return []byte(ret), err
		}
		counter++
		time.Sleep(5 * time.Second)
	}
	return []byte(ret), err
}

func (m *MainActivity) getWiFiClientIPList() ([]byte, error) {
	msg := mp{
		"type":  "get_clients",
		"dummy": "dummy",
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		m.l.Error("getWiFiClientIPList: Error while marshalling get_clients messages")
		return nil, err
	}
	ipaddr := net.ParseIP(m.routerip)
	if ipaddr == nil {
		m.l.Error("getWiFiClientIPList: Error while converting ip address from string to ipaddress")
		return nil, err
	}
	vpnIp := iputil.Ip2VpnIp(ipaddr)
	if err != nil {
		m.l.Error(err.Error() + " getWiFiClientIPList: Error while getting client list")
		return nil, err
	}

	var clients messages.ClientsInfoMessage
	ret, err := m.SendNonTunMessage(vpnIp, jsonData)
	if err != nil {
		m.l.Error(err.Error() + " Message Not sent")
		return nil, err
	}
	err = json.Unmarshal([]byte(ret), &clients)
	if err != nil {
		m.l.Error(err.Error() + " Error while unmarshalling clientsinfomessage " + string(ret))
		return nil, err
	}
	iplist := make([]screen.IPListEntry, len(clients.Clients))
	for i := 0; i < len(clients.Clients); i++ {
		iplist[i].IPAddress = clients.Clients[i].IPAddress
		iplist[i].Name = clients.Clients[i].Name
	}
	msg = mp{
		"iplist": iplist,
	}
	jsonData, err = json.Marshal(msg)
	if err != nil {
		m.l.Error("getWiFiClientIPList: Error while marshalling iplist")
		return nil, err
	}
	return jsonData, nil
}

func (m *MainActivity) getRouterIP() ([]byte, error) {
	var reply RouterIPReply

	bytes, err := m.getIPList(routeripsurl)
	if err != nil {
		m.l.Error("getRouterIP: Error while getting IP List" + err.Error())
		return nil, err
	}
	err = json.Unmarshal(bytes, &reply)
	if err != nil {
		m.l.Error(err.Error() + " getRouterIP: Error while unmarhsalling data")
		return nil, err
	}
	routerip := ""
	if len(reply.Message) == 1 {
		routerip = reply.Message[0].DeviceIp
	} else {
		for i := 0; i < len(reply.Message); i++ {
			if reply.Message[i].PrimaryRouter != 0 {
				routerip = reply.Message[i].DeviceIp
				break
			}
		}
	}
	m.routerip = routerip
	return ([]byte)(routerip), nil
}

func (m *MainActivity) getMobileIPs() ([]string, error) {
	var ips []string
	var reply MobileIPReply

	bytes, err := m.getIPList(mobileipsurl)
	if err != nil {
		m.l.Error("getMobileIPs: Error while getting IP List" + err.Error())
		return nil, err
	}
	err = json.Unmarshal(bytes, &reply)
	if err != nil {
		m.l.Error(err.Error() + " getMobileIPs: Error while unmarhsalling data")
		return nil, err
	}

	if reply.Status != 1 {
		m.l.Error(" getMobileIPs: Not a successful fetch")
		return nil, fmt.Errorf("getMobileIPs: Not a successful fetch")
	}
	ips = make([]string, len(reply.Message))
	for i := 0; i < len(reply.Message); i++ {
		ips[i] = reply.Message[i].DeviceIp
	}
	return ips, nil
}

func (m *MainActivity) processCommands(cmd screen.CommandType, data []byte, length int) ([]byte, error) {
	// Check the current status
	status := m.getLinkStatus()

	// Some inconsistency beween GUI and engine. Expected as we have 5 seconds period
	// to update the status to GUI periodically
	if cmd == screen.Disconnect && (status == screen.LINK_NOT_STARTED || status == screen.LINK_DISCONNECTED || status == screen.LINK_CONNECTION_IN_PROGRESS) {
		// Not connected. Only allowed trigger is to connect
		return nil, nil
	}
	if cmd == screen.Connect && status == screen.LINK_CONNECTED {
		// Connected. Only allowed trigger is to disconnect
		return nil, nil
	}
	switch cmd {
	case screen.Connect:
		m.trigger <- true
		m.l.WithField("Event ", screen.CommandMap[cmd]).Error("2. Command from the GUI")
	case screen.Disconnect:
		m.trigger <- false
		m.l.WithField("Event ", screen.CommandMap[cmd]).Error("Command from the GUI")
	case screen.OnboardClient:
		m.l.WithField("Event ", screen.CommandMap[cmd]).Error("Command from the GUI")
		var onboardData OnboardData

		err := json.Unmarshal(data, &onboardData)
		if err != nil {
			return nil, err
		}
		certs, err := nebula.Sign_nh_client_certs(onboardData.Email, onboardData.Key, onboardData.Name, m.publicKey)
		if err != nil {
			m.l.Error("Error while onboarding. Cannot sign certificates.", err)
			return nil, err
		}

		if certs.ErrorMessage != "" || certs.Cert == "" || certs.Ca == "" || certs.Token == "" {
			m.l.Error("Error while onboarding ", certs.ErrorMessage)
			return nil, fmt.Errorf(certs.ErrorMessage)
		}
		if certs.ErrorMessage == "" && (certs.Cert == "" || certs.Ca == "") {
			m.l.Error("Not able to get the certificates. Contact support.")
			return nil, fmt.Errorf("Not able to get the certificates. Contact support.", certs.ErrorMessage)
		}
		err = m.generate_config(certs)
		if err != nil {
			return nil, err
		}
		m.status_err = ""
		m.trigger <- true
		m.onboarded = true
		m.mw.Onboarded(m.deviceip)
	case screen.UploadLogs:
		err := m.uploadLogs()
		if err != nil {
			return nil, err
		} else {
			return nil, nil
		}
	case screen.ResetConfig:
		m.resetConfig()
	case screen.StopMain:
		m.stopMain()
	case screen.SaveConfig:
		var configData ConfigData

		err := json.Unmarshal(data, &configData)
		if err != nil {
			return nil, err
		}
		return nil, m.saveConfig(configData)
	case screen.GetRouterIP:
		return m.getRouterIP()
	case screen.GetWiFiClientIPList:
		return m.getWiFiClientIPList()
	}
	return nil, nil
}

func (m *MainActivity) Main() {
	var err error

	m.ctrl, m.rs, err = nebula.Main(m.config, false, "Nearhop", m.l, nil)
	if err != nil {
		m.status_err = err.Error()
	}

	switch v := err.(type) {
	case util.ContextualError:
		v.Log(m.l)
	case error:
		m.l.WithError(err).Error("Failed to start")
	}
}

func (m *MainActivity) addHostsEntry(hosts []nebula.ControlHostInfo, connected bool) {
	for _, host := range hosts {
		ip := nh_util.Ip2int(host.VpnIp)
		ne := &screen.NetworkEntry{
			VpnIp:     ip,
			Relay:     host.Relay,
			Connected: connected,
			In_bytes:  host.In_bytes,
			Out_bytes: host.Out_bytes,
			Name:      host.Name,
		}
		m.hosts[ip] = ne
		// Compute Rate
		if m.prevhosts != nil {
			ne1 := m.prevhosts[ip]
			if ne1 != nil {
				ne.In_diff_bytes = (ne.In_bytes - ne1.In_bytes)
				ne.Out_diff_bytes = (ne.Out_bytes - ne1.Out_bytes)
			}
		}
	}
}

func (m *MainActivity) getLinkStatus() screen.LinkStatusType {
	// Check if we started or not
	if m.ctrl == nil {
		return screen.LINK_NOT_STARTED
	}

	hosts := m.ctrl.ListHostmap(false)
	pendinghosts := m.ctrl.ListHostmap(true)

	if hosts == nil && pendinghosts != nil && len(pendinghosts) > 0 {
		return screen.LINK_CONNECTION_IN_PROGRESS
	} else if hosts != nil && len(hosts) > 0 {
		for id := range m.prevhosts {
			delete(m.prevhosts, id)
		}
		for id, value := range m.hosts {
			m.prevhosts[id] = value
			delete(m.hosts, id)
		}
		if hosts != nil {
			m.addHostsEntry(hosts, true)
		}
		if pendinghosts != nil {
			m.addHostsEntry(pendinghosts, false)
		}
		return screen.LINK_CONNECTED
	} else {
		return screen.LINK_DISCONNECTED
	}
}

func (m *MainActivity) setStatus(status string) {
	m.status_err = status
	hd := &screen.HomeDetails{
		Status:      m.Status,
		Name:        m.Name,
		Version:     m.Version,
		CurVersion:  m.CurVersion,
		MyVPNIP:     m.MyVPNIP,
		RelayHostIP: m.RelayHostIP,
		LastUpdated: m.LastUpdated,
		MiscStatus:  m.status_err,
		Hosts:       m.hosts,
	}
	m.mw.SetHomeDetails(hd)
}

// A return value of true indicates the event is not sent to the mobile and hence it is still active
func (m *MainActivity) handleEvent(event *router.RouterEvent) bool {
	fail := true
	for _, ip := range m.MobileIPs {
		jsonData, err := m.rs.GetRouterEventMessage(event)
		if err != nil {
			m.l.Error("handleEvent: Error while getting router event message")
			continue
		}

		ipaddr := net.ParseIP(ip)
		if ipaddr == nil {
			m.l.Error("handleEvent: Error while converting ip address from string to ipaddress")
			continue
		}
		vpnIp := iputil.Ip2VpnIp(ipaddr)
		_, err = m.ctrl.SendNonTunMessage(vpnIp, jsonData)
		if err != nil {
			m.l.Error("handleEvent: Error while sending router message to mobile app ", vpnIp)
		} else {
			fail = false
			m.l.Error("Successfully sent Event data to mobile...", string(jsonData))
		}
	}
	return fail
}

func (m *MainActivity) handleEvents() {
	if !m.onboarded {
		return
	}
	if len(m.MobileIPs) == 0 {
		var err error
		m.MobileIPs, err = m.getMobileIPs()
		if err != nil {
			m.l.Error("handleEvents: Error while getting mobileIPs " + err.Error())
			return
		}
	}
	event := m.rs.GetNextEvent()
	if event != nil {
		ret := m.handleEvent(event)
		m.rs.MarkRouterEvent(event, ret)
	}
}

func (m *MainActivity) Run(ctx context.Context, status_err string) {
	m.status_err = status_err
	if len(status_err) == 0 && m.onboarded {
		// Start connecting
		m.Main()
		if m.ctrl != nil {
			m.ctrl.Start()
			go m.ctrl.ShutdownBlock()
		}
	}

	clockSource := time.NewTicker(5 * time.Second)
	defer clockSource.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case triggered := <-m.trigger:
			if triggered {
				err := m.config.Load(m.configPath)
				if err != nil {
					m.setStatus("failed to load config:" + err.Error())
				}
				m.Main()
				if m.ctrl == nil {
					m.setStatus(m.status_err)
				} else {
					m.ctrl.Start()
					go m.ctrl.ShutdownBlock()
				}
			} else {
				m.stop()
			}
		case tm := <-clockSource.C:
			if m.ctrl != nil {
				m.Status = m.getLinkStatus()
				if m.Name == "" && m.ctrl != nil {
					m.Name = m.ctrl.GetName()
				}
				if m.MyVPNIP == "" && m.ctrl != nil {
					m.MyVPNIP = m.ctrl.GetMyVPNIP()
				}
				if m.ctrl != nil {
					m.RelayHostIP = m.ctrl.GetRelayHostIP()
				}
			}
			m.LastUpdated = tm
			if !m.onboarded {
				m.status_err = ""
			}
			m.setStatus(m.status_err)
			m.handleEvents()
			if m.onboarded && m.rs.ShallUploadLogs() {
				m.uploadLogs()
			}
			if m.onboarded && m.CurVersion == "" {
				m.getDeviceStatus()
			}
		}
	}
}

func (m *MainActivity) stop() {
	if m.ctrl != nil {
		m.ctrl.Stop()
	}
}
