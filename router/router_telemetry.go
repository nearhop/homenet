//go:build router
// +build router

package router

import (
	"container/ring"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	messages "messages"
	nh_util "nh_util"

	"github.com/sirupsen/logrus"
)

const blocklisturl = "https://nearhop.sfo3.digitaloceanspaces.com/blockedips/blacklistips.txt"
const CLIENT_NAME_MAX_LENGTH = 32

type Telemetry struct {
	sync.RWMutex          //Because we concurrently read and write to our maps
	RouterClients         map[string]*RouterClient
	l                     *logrus.Logger
	Repeaters             map[string]*Repeater
	blocklistips          map[string]bool
	NewRepeater           *Repeater
	blocklistupdated      int64
	blockedurllistupdated int64
	clientsdumped         int64
	EventRing             *ring.Ring
}

func readClientDetails(callback addClient, l *logrus.Logger) error {
	files, _ := ioutil.ReadDir(DB_CLIENTS_LOCATION)
	for _, file := range files {
		// Stats and clients are two different files
		filename := getFileName(file.Name())

		content, err := nh_util.NH_read_file(filename)
		if err == nil {
			var client RouterClient
			err = json.Unmarshal(content, &client)
			if err == nil {
				callback(&client)
			}
		} else {
			l.Error("Error", err)
		}
	}
	return nil
}

func dumpClientStats(client *RouterClient) error {
	// Marshal client details
	c, err := json.Marshal(*client)
	if err != nil {
		return err
	}
	curtime := time.Now().Unix()

	nh_util.NH_create_dir(DB_CLIENTS_LOCATION, 0755)
	filename := getFileName(client.MACAddress)
	if curtime-client.Lastseen < (24 * 7 * 3600) {
		return nh_util.NH_dump_to_file(filename, c, 0644)
	} else {
		// More than 7 days, Remove this entry
		return os.Remove(filename)
		return nil
	}

}

func NewTelemetry(l1 *logrus.Logger) *Telemetry {
	t := Telemetry{
		RouterClients: make(map[string]*RouterClient),
		Repeaters:     make(map[string]*Repeater),
		l:             l1,
		EventRing:     ring.New(MAX_NUMBER_OF_EVENTS),
	}
	readClientDetails(func(client *RouterClient) {
		t.RouterClients[client.MACAddress] = client
		t.RouterClients[client.MACAddress].Dirty = false
		addDNSEntryPlatform(client)
	}, l1)
	t.readRepeaterInfo()
	t.getBlockListIPs()
	go t.updateBlockedURLs()
	t.updateClientsList()
	applyDNSEntries()
	return &t
}

func NewRouterClient(mac string, ip string, name string, isrepeater bool, fwver string) *RouterClient {
	rc := RouterClient{
		MACAddress: mac,
		IsRepeater: isrepeater,
		IPAddress:  ip,
		Name:       name,
		Fwver:      fwver,
	}
	return &rc
}

func (tel *Telemetry) getBlockListIPs() {
	bytes, err := nh_util.NH_http_get_req(blocklisturl)
	if err != nil {
		tel.l.Error("Error while fetching blocklist ips")
		tel.blocklistupdated = 0
		return
	}
	entries := strings.Split(string(bytes), "\n")
	if entries == nil {
		tel.l.Error("Error while parsing blocklist ips")
		tel.blocklistupdated = 0
		return
	}
	tel.blocklistips = make(map[string]bool)
	for _, entry := range entries {
		tel.blocklistips[entry] = true
	}
	tel.l.Info("Updated Blocklist ips")
	tel.blocklistupdated = time.Now().Unix()
}

func (tel *Telemetry) updateBlockedURLs() {
	args := []string{""}
	nh_util.NH_read_cmd_output(updateBlockedURLsScript, args)
	tel.blockedurllistupdated = time.Now().Unix()
}

func (tel *Telemetry) dumpRouterClients() {
	tel.RLock()
	defer tel.RUnlock()
	ts1 := time.Now()
	curtime := time.Now().Unix()
	for mac, client := range tel.RouterClients {
		if mac != client.MACAddress {
			// same client with two different MAC Addresses, say repeater
			continue
		}
		if client.Dirty || curtime-client.Lastseen > (24*7*3600) {
			err := dumpClientStats(client)
			client.Dirty = false
			if err != nil {
				tel.l.WithField("mac=", client.MACAddress).Error("Error while dumping client stats", err)
			}
		}
	}
	ts2 := time.Now()
	tel.l.WithField("ts1", ts1).WithField("ts2", ts2).Error("Timestamps")
}

func (tel *Telemetry) Run(ctx context.Context) {
	clockSource := time.NewTicker(30 * time.Second)
	defer clockSource.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-clockSource.C:
			curtime := time.Now().Unix()
			if curtime-tel.clientsdumped >= 3600 {
				// 60 minutes.
				tel.dumpRouterClients()
				tel.clientsdumped = curtime
			}
			if curtime-tel.blocklistupdated >= 24*60*60 {
				// It is 24 hours since we updated the blocklist ips
				// Update now
				tel.getBlockListIPs()
			}
			if curtime-tel.blockedurllistupdated >= 24*60*60 {
				go tel.updateBlockedURLs()
			}
		}
	}
}

func get_host_name(mac string) string {
	args := []string{mac}
	return nh_util.NH_read_cmd_output(get_hostname_cmd, args)
}

func (tel *Telemetry) createClient(mac string, ip string, isrepeater bool, extramac string, fwver string) error {
	if tel.RouterClients[mac] == nil {
		// The device does n't exist. Create one
		multicast, err := nh_util.NH_is_multicast_mac(mac)
		if err != nil {
			return err
		}
		if multicast {
			return fmt.Errorf("Multilcast mac..")
		}
		if ip == "0.0.0.0" || ip == "" {
			return fmt.Errorf("Invalid IP Address for a client")
		}
		name := get_host_name(mac)
		name = strings.Replace(name, "-", "_", -1)
		if len(name) > CLIENT_NAME_MAX_LENGTH {
			name = name[0:CLIENT_NAME_MAX_LENGTH]
		}
		rc := NewRouterClient(mac, ip, name, isrepeater, fwver)
		tel.RouterClients[mac] = rc
		tel.RouterClients[mac].Dirty = true
		tel.newEvent(NEWCLIENTEVENT, ip, tel.RouterClients[mac])
		// Add a domain name entry
		addDNSEntryPlatform(tel.RouterClients[mac])
		applyDNSEntries()
	} else {
		tel.RouterClients[mac].Fwver = fwver
	}
	if extramac != "" {
		tel.RouterClients[extramac] = tel.RouterClients[mac]
		tel.RouterClients[extramac].IsRepeater = isrepeater
	}
	return nil
}

func (tel *Telemetry) anyIPBlockListed(device Device) (bool, string) {
	for _, conn := range device.Conn {
		if tel.blocklistips[conn.R_ip] == true {
			return true, conn.R_ip
		}
	}
	return false, ""
}

func (tel *Telemetry) newEvent(etype EventType, ip string, client *RouterClient) {
	event := &RouterEvent{
		Etype:  etype,
		Extra:  ip,
		Client: client,
		Active: true,
		Tstamp: time.Now().Unix(),
	}
	tel.EventRing.Value = event
	tel.EventRing = tel.EventRing.Next()
}

func (tel *Telemetry) getNextEvent() *RouterEvent {
	tel.Lock()
	defer tel.Unlock()

	var ev *RouterEvent
	ev = nil

	tel.EventRing.Do(func(p interface{}) {
		if p != nil && ev == nil {
			event := p.(*RouterEvent)
			if event.Active {
				ev = event
			}
		}
	})
	return ev
}

func (tel *Telemetry) processTelemetry(telemetryData *TelemetryData) error {
	tel.Lock()
	defer tel.Unlock()
	for _, device := range telemetryData.Devices {
		blocked, ip := tel.anyIPBlockListed(device)
		if blocked {
			if tel.RouterClients[device.Mac] != nil {
				tel.newEvent(BLOCKEDIPEVENT, ip, tel.RouterClients[device.Mac])
			}
			tel.l.Error("Traffic to Blocklisted IP")
		}
		err := tel.createClient(device.Mac, device.Ip, false, "", "")
		if err != nil {
			tel.l.WithField("Client MAC Address", device.Mac).Error(err.Error())
		}
		if tel.RouterClients[device.Mac] != nil {
			tel.RouterClients[device.Mac].Lastseen = time.Now().Unix()
			tel.RouterClients[device.Mac].IPAddress = device.Ip
		}
	}
	return nil
}

func (tel *Telemetry) updateWireless(sta Station, channel int) error {
	if tel.RouterClients[sta.Mac] == nil {
		return nil
	}
	tel.RouterClients[sta.Mac].Type = WIRELESS
	tel.RouterClients[sta.Mac].Channel = channel
	tel.RouterClients[sta.Mac].Rssi = sta.Rssi
	tel.RouterClients[sta.Mac].Dirty = true
	tel.RouterClients[sta.Mac].Lastseen = time.Now().Unix()
	return nil
}

func (tel *Telemetry) processWirelessTelemetry(wirelessTelemetryData *WirelessTelemetryData) error {
	tel.Lock()
	defer tel.Unlock()
	for _, radio := range wirelessTelemetryData.Radios {
		for _, vap := range radio.Vaps {
			for _, station := range vap.Stas {
				tel.updateWireless(station, radio.Channel)
			}
		}
	}
	return nil
}

func (tel *Telemetry) readRepeaterInfo() {
	repjson, err := nh_util.NH_read_file(DB_REPEATERS_FILE)
	if err != nil {
		return
	}
	repeaters := make([]Repeater, MAX_NUM_OF_REPEATERS)
	if repeaters == nil {
		tel.l.Error("Error while allocating repeaters memory")
		return
	}
	err = json.Unmarshal(repjson, &repeaters)
	if err != nil {
		tel.l.Error("Error while unmarshalling repeater info")
		return
	}
	for i := 0; i < MAX_NUM_OF_REPEATERS; i++ {
		tel.Repeaters[repeaters[i].Mac] = &repeaters[i]
	}
}

func (tel *Telemetry) RepeatersJson() ([]byte, error) {
	// Copy into repeaters do dump into db
	repeaters := make([]Repeater, MAX_NUM_OF_REPEATERS)
	if repeaters == nil {
		tel.l.Error(nh_util.NH_getErrorStatusString("Error while allocating repeaters memory"))
		return nil, fmt.Errorf("Error while allocating repeaters memory")
	}

	i := 0
	for _, repeater := range tel.Repeaters {
		if i > MAX_NUM_OF_REPEATERS {
			return nil, fmt.Errorf("How come there are more repeaters than permitted")
		}
		repeaters[i].Mac = repeater.Mac
		repeaters[i].MMac = repeater.MMac
		repeaters[i].Ip = repeater.Ip
		repeaters[i].Name = repeater.Name
		repeaters[i].Fwver = repeater.Fwver
		i++
	}
	rbytes, err := json.Marshal(repeaters)
	if err != nil {
		return nil, err
	}
	return rbytes, nil
}

func (tel *Telemetry) registerRepeater(data []byte) error {
	var repeaterMessage RepeaterMessage
	err := json.Unmarshal(data, &repeaterMessage)
	if err != nil {
		return err
	}
	repeater := &Repeater{
		Name:  repeaterMessage.Name,
		Mac:   repeaterMessage.Mac,
		MMac:  repeaterMessage.MMac,
		Ip:    repeaterMessage.Ip,
		Fwver: repeaterMessage.Fwver,
	}
	if len(tel.Repeaters) >= MAX_NUM_OF_REPEATERS {
		tel.l.Error(nh_util.NH_getErrorStatusString("Maximum number of repeater reached"))
		return fmt.Errorf("Maximum number of repeater reached")
	}
	if tel.Repeaters[repeater.Mac] == nil {
		// New Repeater
		tel.NewRepeater = repeater
	}
	tel.Repeaters[repeater.Mac] = repeater

	rbytes, err := tel.RepeatersJson()
	if err != nil {
		return err
	}
	err = nh_util.NH_dump_to_file(DB_REPEATERS_FILE, rbytes, 0644)
	if err != nil {
		return err
	}
	err = tel.createClient(repeater.Mac, repeater.Ip, true, repeaterMessage.MMac, repeaterMessage.Fwver)
	if err != nil {
		tel.l.WithField("Client MAC Address", repeater.Mac).Error(err.Error())
		return err
	}
	// Dump the clients into Json
	tel.dumpRouterClients()
	return nil
}

func (tel *Telemetry) pauseClient(mac string, pause bool) string {
	tel.Lock()
	defer tel.Unlock()
	if tel.RouterClients[mac] == nil {
		tel.l.Error("Received a pause/unpause request for a client that does n't exist")
		return nh_util.NH_getErrorStatusString("Received a pause/unpause request for a client that does n't exist")
	}
	tel.RouterClients[mac].Paused = pause
	err := dumpClientStats(tel.RouterClients[mac])
	if err != nil {
		tel.l.Error("Error while saving (pause) the client details", tel.RouterClients[mac].MACAddress)
		return nh_util.NH_getErrorStatusString("Error while saving the client details")
	}
	ip := tel.RouterClients[mac].IPAddress
	return pauseClient(mac, ip, tel.RouterClients[mac].Name, pause)
}

func (tel *Telemetry) pauseAll(pause bool) string {
	tel.RLock()
	defer tel.RUnlock()

	for _, client := range tel.RouterClients {
		client.Paused = pause
		err := dumpClientStats(client)
		if err != nil {
			tel.l.Error("Error while saving (pauseall) the client details", client.MACAddress)
			return nh_util.NH_getErrorStatusString("Error while saving the client details")
		}
	}
	return pauseAll(pause)
}

func (tel *Telemetry) dumpClientsJson() ([]byte, error) {
	tel.RLock()
	defer tel.RUnlock()

	clients := make([]messages.ClientInfo, messages.MAX_NUM_OF_CLIENTS)
	index := 0
	curtime := time.Now().Unix()
	for mac, client := range tel.RouterClients {
		if mac != client.MACAddress {
			// same client with two different MAC Addresses, say repeater
			continue
		}
		if curtime-client.Lastseen > (24 * 3600) {
			// Not active for more than a day
			continue
		}
		clients[index].MACAddress = client.MACAddress
		clients[index].IPAddress = client.IPAddress
		clients[index].Name = client.Name
		clients[index].Type = int(client.Type)
		clients[index].Channel = client.Channel
		clients[index].SignalQuality = getSignalQuality(client.Rssi)
		clients[index].Paused = client.Paused
		clients[index].IsRepeater = client.IsRepeater
		clients[index].Lastseen = client.Lastseen
		index++
	}
	jsonData, err := json.Marshal(clients[0:index])
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func (tel *Telemetry) setClientDetails(mac string, name string, Type uint8) string {
	tel.Lock()
	defer tel.Unlock()
	if tel.RouterClients[mac] == nil {
		tel.l.Error("Received a setClient request for a client that does n't exist")
		return nh_util.NH_getErrorStatusString("Received a setClient request for a client that does n't exist")
	}
	tel.RouterClients[mac].Name = name
	tel.RouterClients[mac].Type = (ClientType)(Type)
	err := dumpClientStats(tel.RouterClients[mac])
	if err != nil {
		tel.l.Error("Error while saving the client details", tel.RouterClients[mac].MACAddress)
		return nh_util.NH_getErrorStatusString("Error while saving the client details")
	}

	tel.updateClientsList()
	addDNSEntryPlatform(tel.RouterClients[mac])
	applyDNSEntries()

	return ""
}

func (tel *Telemetry) updateClientsList() {
	out := ""
	for mac, client := range tel.RouterClients {
		if mac != client.MACAddress {
			// same client with two different MAC Addresses, say repeater
			continue
		}
		out = out + client.IPAddress + " " + client.Name + "\n"
	}
	nh_util.NH_dump_to_file(nearhop_hostnames, ([]byte)(out), 0744)
}
