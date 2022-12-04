package nebula

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

const (
	DefaultHandshakeTryInterval   = time.Millisecond * 1000
	DefaultHandshakeRetries       = 5
	DefaultHandshakeTriggerBuffer = 64
)

var (
	defaultHandshakeConfig = HandshakeConfig{
		tryInterval:   DefaultHandshakeTryInterval,
		retries:       DefaultHandshakeRetries,
		triggerBuffer: DefaultHandshakeTriggerBuffer,
	}
)

type HandshakeConfig struct {
	tryInterval   time.Duration
	retries       int
	triggerBuffer int

	messageMetrics *MessageMetrics
}

type HandshakeManager struct {
	pendingHostMap         *HostMap
	mainHostMap            *HostMap
	lightHouse             *LightHouse
	outside                *udp.Conn
	config                 HandshakeConfig
	OutboundHandshakeTimer *SystemTimerWheel
	messageMetrics         *MessageMetrics
	metricInitiated        metrics.Counter
	metricTimedOut         metrics.Counter
	l                      *logrus.Logger
	networkID              uint64

	// can be used to trigger outbound handshake for the given vpnIp
	trigger chan NetworkIPPair
}

func NewHandshakeManager(l *logrus.Logger, tunCidr *net.IPNet, preferredRanges []*net.IPNet, mainHostMap *HostMap, lightHouse *LightHouse, outside *udp.Conn, config HandshakeConfig, networkID uint64) *HandshakeManager {
	l.Error("Hostmap will be created", config.tryInterval, hsTimeout(config.retries, config.tryInterval), config.retries, config.tryInterval)
	return &HandshakeManager{
		pendingHostMap:         NewHostMap(l, "pending", tunCidr, preferredRanges),
		mainHostMap:            mainHostMap,
		lightHouse:             lightHouse,
		outside:                outside,
		config:                 config,
		trigger:                make(chan NetworkIPPair, config.triggerBuffer),
		OutboundHandshakeTimer: NewSystemTimerWheel(90*time.Millisecond, hsTimeout(config.retries, config.tryInterval)),
		messageMetrics:         config.messageMetrics,
		metricInitiated:        metrics.GetOrRegisterCounter("handshake_manager.initiated", nil),
		metricTimedOut:         metrics.GetOrRegisterCounter("handshake_manager.timed_out", nil),
		l:                      l,
		networkID:              networkID,
	}
}

func (c *HandshakeManager) Run(ctx context.Context, f udp.EncWriter) {
	clockSource := time.NewTicker(110 * time.Millisecond)
	defer clockSource.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case nip := <-c.trigger:
			c.l.WithField("vpnIp", nip.vpnIP).WithField("networkID", nip.networkID).Info("HandshakeManager: triggered")
			c.handleOutbound(nip.vpnIP, f, true, nip.networkID)
		case now := <-clockSource.C:
			//c.l.Info("Tmeout. HandshakeManager: triggered")
			c.NextOutboundHandshakeTimerTick(now, f)
		}
	}
}

func (c *HandshakeManager) NextOutboundHandshakeTimerTick(now time.Time, f udp.EncWriter) {
	c.OutboundHandshakeTimer.advance(now)
	for {
		ep := c.OutboundHandshakeTimer.Purge()
		if ep == nil {
			break
		}
		nip := ep.(NetworkIPPair)

		c.l.WithField("vpnIp", nip.vpnIP).Info("HandshakeManager: Purged")
		c.handleOutbound(nip.vpnIP, f, false, nip.networkID)
	}
}

func (c *HandshakeManager) handleOutbound(vpnIp iputil.VpnIp, f udp.EncWriter, lighthouseTriggered bool, networkID uint64) {
	hostinfo, err := c.pendingHostMap.QueryVpnIp(vpnIp, networkID)
	if err != nil {
		//c.l.WithField("lighthouseTriggered ", lighthouseTriggered).WithField("vpnIp", vpnIp).WithField("networkID", networkID).Error("No hostinfo")
		return
	}
	hostinfo.Lock()
	defer hostinfo.Unlock()

	hostinfo.logger(c.l).WithField("hostinfo.HandshakeComplete", hostinfo.HandshakeComplete).WithField("lighthouseTriggered ", lighthouseTriggered).WithField("hostinfo.HandshakeCounter", hostinfo.HandshakeCounter).Info("##########################")
	// We may have raced to completion but now that we have a lock we should ensure we have not yet completed.
	if hostinfo.HandshakeComplete {
		// Ensure we don't exist in the pending hostmap anymore since we have completed
		c.pendingHostMap.DeleteHostInfo(hostinfo)
		return
	}
	nip := NetworkIPPair{
		vpnIP:     vpnIp,
		networkID: networkID,
	}

	// Check if we have a handshake packet to transmit yet
	if !hostinfo.HandshakeReady {
		// There is currently a slight race in getOrHandshake due to ConnectionState not being part of the HostInfo directly
		// Our hostinfo here was added to the pending map and the wheel may have ticked to us before we created ConnectionState
		c.l.WithField("vpnIp", vpnIp).Error("HandshakeManager: HandshakeNotready")
		c.OutboundHandshakeTimer.Add(nip, c.config.tryInterval)
		return
	}

	// If we are out of time, clean up
	if hostinfo.HandshakeCounter >= c.config.retries && !c.lightHouse.IsLighthouseIP(hostinfo.vpnIp) {
		// if hostinfo.HandshakeCounter >= c.config.retries {
		// To force relay, use the following
		// if !c.lightHouse.amLighthouse && !c.lightHouse.IsLighthouseIP(hostinfo.vpnIp) {
		// Get a remotes object if we don't already have one.
		// This is mainly to protect us as this should never be the case
		if hostinfo.remotes == nil || len(hostinfo.remotes.addrs) == 0 {
			c.l.WithField("vpnIp", vpnIp).Info("HandshakeManager: Remoteslist empty")
			c.metricTimedOut.Inc(1)
			c.pendingHostMap.DeleteHostInfo(hostinfo)
			return
		}
		if hostinfo.HandshakeCounter == 2*c.config.retries {
			// The current relay server is not helping
			// Switch to another relay server
			hostinfo.relayIP = f.GetABetterRelayServer()
			if hostinfo.relayIP != nil {
				hostinfo.logger(c.l).WithField("hostinfo.relayIP", hostinfo.relayIP).WithField("hostinfo.vpnIp", hostinfo.vpnIp).Info("Using a better Relay server for")
			}
		}
		// No response for handshake
		// Check if relaying helps
		hostinfo.logger(c.l).WithField("udpAddrs", hostinfo.remotes.CopyAddrs(c.pendingHostMap.preferredRanges)).
			WithField("initiatorIndex", hostinfo.localIndexId).
			WithField("remoteIndex", hostinfo.remoteIndexId).
			WithField("handshake", m{"stage": 1, "style": "ix_psk0"}).
			WithField("durationNs", time.Since(hostinfo.handshakeStart).Nanoseconds()).
			Info("Handshake timed out. Trying Relay server.")

		hostinfo.networkID = networkID
		err := f.SendRelay(header.RelayPacket, 0, hostinfo.HandshakePacket[0], make([]byte, 12, 12), make([]byte, mtu), (uint32)(vpnIp), udp.Ip2int(c.mainHostMap.vpnCIDR.IP), 0, 0, hostinfo.networkID, hostinfo.relayIP)
		if err != nil {
			hostinfo.logger(c.l).WithField("udpAddrs", hostinfo.remotes.CopyAddrs(c.pendingHostMap.preferredRanges)).
				WithField("initiatorIndex", hostinfo.localIndexId).
				WithField("remoteIndex", hostinfo.remoteIndexId).
				WithField("handshake", m{"stage": 1, "style": "ix_psk0"}).
				WithField("durationNs", time.Since(hostinfo.handshakeStart).Nanoseconds()).
				Info("Error while trying to send handshake packet through Relay server")
			return
		}
		hostinfo.relay = 1
		// Increment the counter to increase our delay, linear backoff
		hostinfo.HandshakeCounter++
		// Clearout the entries only if the relaying also fails
		if hostinfo.HandshakeCounter >= 3*c.config.retries {
			c.metricTimedOut.Inc(1)
			c.pendingHostMap.DeleteHostInfo(hostinfo)
			return
		}
		// If a lighthouse triggered this attempt then we are still in the timer wheel and do not need to re-add
		if !lighthouseTriggered {
			//TODO: feel like we dupe handshake real fast in a tight loop, why?
			c.l.WithField("vpnIp", vpnIp).Info("HandshakeManager: Added1")
			// Relay server. Double the try interval
			c.OutboundHandshakeTimer.Add(nip, 2*c.config.tryInterval)
		}
		return
	}
	if hostinfo.HandshakeCounter >= c.config.retries {
		c.metricTimedOut.Inc(1)
		c.pendingHostMap.DeleteHostInfo(hostinfo)
	}

	// We only care about a lighthouse trigger before the first handshake transmit attempt. This is a very specific
	// optimization for a fast lighthouse reply
	//TODO: it would feel better to do this once, anytime, as our delay increases over time
	if lighthouseTriggered && hostinfo.HandshakeCounter > 0 {
		// If we didn't return here a lighthouse could cause us to aggressively send handshakes
		return
	}

	// Get a remotes object if we don't already have one.
	// This is mainly to protect us as this should never be the case
	if hostinfo.remotes == nil {
		hostinfo.remotes = c.lightHouse.QueryCache(vpnIp, c.networkID)
	}

	//TODO: this will generate a load of queries for hosts with only 1 ip (i'm not using a lighthouse, static mapped)
	if hostinfo.remotes != nil && hostinfo.remotes.Len(c.pendingHostMap.preferredRanges) <= 1 {
		// If we only have 1 remote it is highly likely our query raced with the other host registered within the lighthouse
		// Our vpnIp here has a tunnel with a lighthouse but has yet to send a host update packet there so we only know about
		// the learned public ip for them. Query again to short circuit the promotion counter
		c.lightHouse.QueryServer(vpnIp, networkID, f)
	}

	// Send a the handshake to all known ips, stage 2 takes care of assigning the hostinfo.remote based on the first to reply
	var sentTo []*udp.Addr
	hostinfo.remotes.ForEach(c.pendingHostMap.preferredRanges, func(addr *udp.Addr, _ bool) {
		c.messageMetrics.Tx(header.Handshake, header.MessageSubType(hostinfo.HandshakePacket[0][1]), 1)
		err = c.outside.WriteTo(hostinfo.HandshakePacket[0], addr)
		if err != nil {
			hostinfo.logger(c.l).WithField("udpAddr", addr).
				WithField("initiatorIndex", hostinfo.localIndexId).
				WithField("handshake", m{"stage": 1, "style": "ix_psk0"}).
				WithError(err).Error("Failed to send handshake message")

		} else {
			sentTo = append(sentTo, addr)
			hostinfo.logger(c.l).WithField("udpAddr", addr).Error("Handshake Message has been sent")
		}
	})

	// Don't be too noisy or confusing if we fail to send a handshake - if we don't get through we'll eventually log a timeout
	if len(sentTo) > 0 {
		hostinfo.logger(c.l).WithField("udpAddrs", sentTo).
			WithField("initiatorIndex", hostinfo.localIndexId).
			WithField("handshake", m{"stage": 1, "style": "ix_psk0"}).
			Error("1. Handshake message sent")
	}

	// Increment the counter to increase our delay, linear backoff
	hostinfo.HandshakeCounter++

	// If a lighthouse triggered this attempt then we are still in the timer wheel and do not need to re-add
	if !lighthouseTriggered {
		//TODO: feel like we dupe handshake real fast in a tight loop, why?
		c.l.WithField("vpnIp", vpnIp).WithField("hostinfo.HandshakeCounter", hostinfo.HandshakeCounter).WithField("c.config.tryInterval", c.config.tryInterval).Error("HandshakeManager: Added22")
		c.OutboundHandshakeTimer.Add(nip, c.config.tryInterval)
	}
}

func (c *HandshakeManager) AddVpnIp(vpnIp iputil.VpnIp, networkID uint64, init func(*HostInfo)) *HostInfo {
	c.l.WithField("vpnIp", vpnIp).WithField("networkID", networkID).Error("HandshakeManager: Added333")
	hostinfo, created := c.pendingHostMap.AddVpnIp(vpnIp, networkID, init)

	nip := NetworkIPPair{
		vpnIP:     vpnIp,
		networkID: networkID,
	}

	if created {
		c.OutboundHandshakeTimer.Add(nip, c.config.tryInterval)
		c.metricInitiated.Inc(1)
	}

	return hostinfo
}

var (
	ErrExistingHostInfo    = errors.New("existing hostinfo")
	ErrAlreadySeen         = errors.New("already seen")
	ErrLocalIndexCollision = errors.New("local index collision")
	ErrExistingHandshake   = errors.New("existing handshake")
)

// CheckAndComplete checks for any conflicts in the main and pending hostmap
// before adding hostinfo to main. If err is nil, it was added. Otherwise err will be:
//
// ErrAlreadySeen if we already have an entry in the hostmap that has seen the
// exact same handshake packet
//
// ErrExistingHostInfo if we already have an entry in the hostmap for this
// VpnIp and the new handshake was older than the one we currently have
//
// ErrLocalIndexCollision if we already have an entry in the main or pending
// hostmap for the hostinfo.localIndexId.
func (c *HandshakeManager) CheckAndComplete(hostinfo *HostInfo, handshakePacket uint8, overwrite bool, f *Interface, networkID uint64) (*HostInfo, error) {
	c.pendingHostMap.Lock()
	defer c.pendingHostMap.Unlock()
	c.mainHostMap.Lock()
	defer c.mainHostMap.Unlock()

	// Check if we already have a tunnel with this vpn ip
	existingHostInfo, found := c.mainHostMap.Hosts[networkID][hostinfo.vpnIp]
	if found && existingHostInfo != nil {
		// Is it just a delayed handshake packet?
		if bytes.Equal(hostinfo.HandshakePacket[handshakePacket], existingHostInfo.HandshakePacket[handshakePacket]) {
			return existingHostInfo, ErrAlreadySeen
		}

		// Is this a newer handshake?
		if existingHostInfo.lastHandshakeTime >= hostinfo.lastHandshakeTime {
			return existingHostInfo, ErrExistingHostInfo
		}

		existingHostInfo.logger(c.l).Info("Taking new handshake")
	}

	existingIndex, found := c.mainHostMap.Indexes[networkID][hostinfo.localIndexId]
	if found {
		// We have a collision, but for a different hostinfo
		return existingIndex, ErrLocalIndexCollision
	}

	existingIndex, found = c.pendingHostMap.Indexes[networkID][hostinfo.localIndexId]
	if found && existingIndex != hostinfo {
		// We have a collision, but for a different hostinfo
		return existingIndex, ErrLocalIndexCollision
	}

	existingRemoteIndex, found := c.mainHostMap.RemoteIndexes[networkID][hostinfo.remoteIndexId]
	if found && existingRemoteIndex != nil && existingRemoteIndex.vpnIp != hostinfo.vpnIp {
		// We have a collision, but this can happen since we can't control
		// the remote ID. Just log about the situation as a note.
		hostinfo.logger(c.l).
			WithField("remoteIndex", hostinfo.remoteIndexId).WithField("collision", existingRemoteIndex.vpnIp).
			Info("New host shadows existing host remoteIndex")
	}

	// Check if we are also handshaking with this vpn ip
	pendingHostInfo, found := c.pendingHostMap.Hosts[networkID][hostinfo.vpnIp]
	if found && pendingHostInfo != nil {
		if !overwrite {
			// We won, let our pending handshake win
			return pendingHostInfo, ErrExistingHandshake
		}

		// We lost, take this handshake and move any cached packets over so they get sent
		if pendingHostInfo.ConnectionState != nil {
			pendingHostInfo.ConnectionState.queueLock.Lock()
		}
		hostinfo.packetStore = append(hostinfo.packetStore, pendingHostInfo.packetStore...)
		c.pendingHostMap.unlockedDeleteHostInfo(pendingHostInfo)
		if pendingHostInfo.ConnectionState != nil {
			pendingHostInfo.ConnectionState.queueLock.Unlock()
		}
		pendingHostInfo.logger(c.l).Info("Handshake race lost, replacing pending handshake with completed tunnel")
	}

	if existingHostInfo != nil {
		// We are going to overwrite this entry, so remove the old references
		delete(c.mainHostMap.Hosts[networkID], existingHostInfo.vpnIp)
		delete(c.mainHostMap.Indexes[networkID], existingHostInfo.localIndexId)
		delete(c.mainHostMap.RemoteIndexes[networkID], existingHostInfo.remoteIndexId)
	}

	hostinfo.networkID = networkID
	if !f.lightHouse.amLighthouse {
		// For normal hosts, only one networkID ie its own id
		networkID = f.networkID
	}
	hostinfo.logger(c.l).WithField("vpnIP", hostinfo.vpnIp).WithField("remoteIndex", hostinfo.remoteIndexId).WithField("networkID", networkID).Error("Going to add host 1")
	c.mainHostMap.addHostInfo(hostinfo, f)
	return existingHostInfo, nil
}

// Complete is a simpler version of CheckAndComplete when we already know we
// won't have a localIndexId collision because we already have an entry in the
// pendingHostMap
func (c *HandshakeManager) Complete(hostinfo *HostInfo, f *Interface, networkID uint64) {
	c.pendingHostMap.Lock()
	defer c.pendingHostMap.Unlock()
	c.mainHostMap.Lock()
	defer c.mainHostMap.Unlock()

	existingHostInfo, found := c.mainHostMap.Hosts[networkID][hostinfo.vpnIp]
	if found && existingHostInfo != nil {
		// We are going to overwrite this entry, so remove the old references
		delete(c.mainHostMap.Hosts[networkID], existingHostInfo.vpnIp)
		delete(c.mainHostMap.Indexes[networkID], existingHostInfo.localIndexId)
		delete(c.mainHostMap.RemoteIndexes[networkID], existingHostInfo.remoteIndexId)
	}

	existingRemoteIndex, found := c.mainHostMap.RemoteIndexes[networkID][hostinfo.remoteIndexId]
	if found && existingRemoteIndex != nil {
		// We have a collision, but this can happen since we can't control
		// the remote ID. Just log about the situation as a note.
		hostinfo.logger(c.l).
			WithField("remoteIndex", hostinfo.remoteIndexId).WithField("collision", existingRemoteIndex.vpnIp).
			Info("New host shadows existing host remoteIndex")
	}

	if !f.lightHouse.amLighthouse {
		// For normal hosts, only one networkID ie its own id
		networkID = f.networkID
	}
	hostinfo.networkID = networkID
	hostinfo.logger(c.l).WithField("vpnIP", hostinfo.vpnIp).WithField("remoteIndex", hostinfo.remoteIndexId).WithField("networkID", networkID).Error("Going to add host 2")
	c.mainHostMap.addHostInfo(hostinfo, f)
	c.pendingHostMap.unlockedDeleteHostInfo(hostinfo)
}

// AddIndexHostInfo generates a unique localIndexId for this HostInfo
// and adds it to the pendingHostMap. Will error if we are unable to generate
// a unique localIndexId
func (c *HandshakeManager) AddIndexHostInfo(h *HostInfo, networkID uint64) error {
	c.pendingHostMap.Lock()
	defer c.pendingHostMap.Unlock()
	c.mainHostMap.RLock()
	defer c.mainHostMap.RUnlock()

	for i := 0; i < 32; i++ {
		index, err := generateIndex(c.l)
		if err != nil {
			return err
		}

		_, inPending := c.pendingHostMap.Indexes[networkID][index]
		_, inMain := c.mainHostMap.Indexes[networkID][index]

		if !inMain && !inPending {
			h.localIndexId = index
			if c.pendingHostMap.Indexes[networkID] == nil {
				c.pendingHostMap.Indexes[networkID] = make(map[uint32]*HostInfo)
			}
			c.pendingHostMap.Indexes[networkID][index] = h
			return nil
		}
	}

	return errors.New("failed to generate unique localIndexId")
}

func (c *HandshakeManager) addRemoteIndexHostInfo(index uint32, h *HostInfo, networkID uint64) {
	c.pendingHostMap.addRemoteIndexHostInfo(index, h, networkID)
}

func (c *HandshakeManager) DeleteHostInfo(hostinfo *HostInfo) {
	//l.Debugln("Deleting pending hostinfo :", hostinfo)
	c.pendingHostMap.DeleteHostInfo(hostinfo)
}

func (c *HandshakeManager) QueryIndex(index uint32, networkID uint64) (*HostInfo, error) {
	return c.pendingHostMap.QueryIndex(index, networkID)
}

func (c *HandshakeManager) EmitStats() {
	c.pendingHostMap.EmitStats("pending")
	c.mainHostMap.EmitStats("main")
}

// Utility functions below

func generateIndex(l *logrus.Logger) (uint32, error) {
	b := make([]byte, 4)

	// Let zero mean we don't know the ID, so don't generate zero
	var index uint32
	for index == 0 {
		_, err := rand.Read(b)
		if err != nil {
			l.Errorln(err)
			return 0, err
		}

		index = binary.BigEndian.Uint32(b)
	}

	if l.Level >= logrus.DebugLevel {
		l.WithField("index", index).
			Debug("Generated index")
	}
	return index, nil
}

func hsTimeout(tries int, interval time.Duration) time.Duration {
	return time.Duration((tries + 1) * int(interval))
}
