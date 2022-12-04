package nebula

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flynn/noise"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/firewall"
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

func (f *Interface) consumeInsidePacket(packet []byte, fwPacket *firewall.Packet, nb, out []byte, q int, localCache firewall.ConntrackCache) {
	err := newPacket(packet, false, fwPacket)
	if err != nil {
		f.l.WithField("packet", packet).Debugf("Error while validating outbound packet: %s", err)
		return
	}

	// Ignore local broadcast packets
	if f.dropLocalBroadcast && fwPacket.RemoteIP == f.localBroadcast {
		return
	}

	// Ignore packets from self to self
	if fwPacket.RemoteIP == f.myVpnIp {
		return
	}

	// Ignore broadcast packets
	if f.dropMulticast && isMulticast(fwPacket.RemoteIP) {
		return
	}

	hostinfo := f.getOrHandshake(fwPacket.RemoteIP, f.networkID, true)
	if hostinfo == nil {
		if f.l.Level >= logrus.DebugLevel {
			f.l.WithField("vpnIp", fwPacket.RemoteIP).
				WithField("fwPacket", fwPacket).
				Debugln("dropping outbound packet, vpnIp not in our CIDR or in unsafe routes")
		}
		return
	}
	ci := hostinfo.ConnectionState

	if ci.ready == false {
		// Because we might be sending stored packets, lock here to stop new things going to
		// the packet queue.
		ci.queueLock.Lock()
		if !ci.ready {
			//hostinfo.cachePacket(f.l, header.Message, 0, packet, f.sendMessageNow, f.cachedPacketMetrics)
			ci.queueLock.Unlock()
			return
		}
		ci.queueLock.Unlock()
	}

	f.sendNoMetrics(header.Message, 0, ci, hostinfo, hostinfo.remote, packet, nb, out, q)
	// Venkat: Revisit this. We had to comment this to remove the static capool
	/*
		dropReason := f.firewall.Drop(packet, *fwPacket, false, hostinfo, f.caPool, localCache)
		if dropReason == nil {

		} else if f.l.Level >= logrus.DebugLevel {
			hostinfo.logger(f.l).
				WithField("fwPacket", fwPacket).
				WithField("reason", dropReason).
				Debugln("dropping outbound packet")
		}
	*/
}

// getOrHandshake returns nil if the vpnIp is not routable
func (f *Interface) getOrHandshake(vpnIp iputil.VpnIp, networkID uint64, initHandshake bool) *HostInfo {
	var vpnmode uint8

	// If I am lighthouse and networkid is 0, I don't need to do any lookup
	if f.lightHouse.amLighthouse && networkID == 0 {
		return nil
	}
	// (f.l).Info("VPNMode set to 1")

	vpnmode = 1
	//TODO: we can find contains without converting back to bytes
	if f.hostMap.vpnCIDR.Contains(vpnIp.ToIP()) == false {
		vpnIp = f.inside.RouteFor(vpnIp)
		if vpnIp == 0 {
			return nil
		}
		// (f.l).Info("VPNMode set to 2")
		vpnmode = 2
	}
	hostinfo, err := f.hostMap.PromoteBestQueryVpnIp(vpnIp, f, networkID)

	if hostinfo != nil && hostinfo.VpnMode != 2 {
		hostinfo.VpnMode = vpnmode
	}

	if !initHandshake {
		return hostinfo
	}
	//if err != nil || hostinfo.ConnectionState == nil {
	if err != nil {
		if f.lightHouse.amLighthouse && !initHandshake {
			// No need proceeed further
			return nil
		}
		hostinfo, err = f.handshakeManager.pendingHostMap.QueryVpnIp(vpnIp, networkID)
		if err != nil {
			hostinfo = f.handshakeManager.AddVpnIp(vpnIp, networkID, f.initHostInfo)
			if hostinfo == nil {
				hostinfo.logger(f.l).Info("hostinfo is nil")
			} else {
				hostinfo.logger(f.l).Info("hostinfo is not nil")
			}
		}
	}
	ci := hostinfo.ConnectionState

	if ci != nil && ci.eKey != nil && ci.ready {
		return hostinfo
	}
	if hostinfo.VpnMode != 2 {
		hostinfo.VpnMode = vpnmode
	}
	// Handshake is not ready, we need to grab the lock now before we start the handshake process
	hostinfo.Lock()
	defer hostinfo.Unlock()

	// Double check, now that we have the lock
	ci = hostinfo.ConnectionState
	if ci != nil && ci.eKey != nil && ci.ready {
		return hostinfo
	}

	// If we have already created the handshake packet, we don't want to call the function at all.
	if !hostinfo.HandshakeReady {
		ixHandshakeStage0(f, vpnIp, hostinfo)
		// FIXME: Maybe make XX selectable, but probably not since psk makes it nearly pointless for us.
		//xx_handshakeStage0(f, ip, hostinfo)

		// If this is a static host, we don't need to wait for the HostQueryReply
		// We can trigger the handshake right now
		if _, ok := f.lightHouse.staticList[vpnIp]; ok {
			nip := NetworkIPPair{
				vpnIP:     vpnIp,
				networkID: networkID,
			}
			select {
			case f.handshakeManager.trigger <- nip:
				hostinfo.logger(f.l).WithField("vpnIp", udp.Int2ip((uint32)(vpnIp))).Info("Triggered")
			default:
			}
		}
	}

	return hostinfo
}

// initHostInfo is the init function to pass to (*HandshakeManager).AddVpnIP that
// will create the initial Noise ConnectionState
func (f *Interface) initHostInfo(hostinfo *HostInfo) {
	var certState *CertState
	var err error
	certState = f.certState
	if f.lightHouse.amLighthouse {
		err = fmt.Errorf("Signing the certificates under progress")
		sendSignRequest := shallSendSignRequest(f, hostinfo.networkID)
		if f.certStateLock[hostinfo.networkID] == nil {
			f.certStateLock[hostinfo.networkID] = &sync.RWMutex{}
		}
		f.certStateLock[hostinfo.networkID].Lock()
		defer f.certStateLock[hostinfo.networkID].Unlock()
		certState, err = getrawCertState(hostinfo.networkID, f.relayIndex, f.sqlsecret, f.keysecret, sendSignRequest)
		if err != nil {
			f.l.WithError(err).WithField("networkID", hostinfo.networkID).Error("Error with initHostInfo")
		}
	}
	if certState != nil {
		hostinfo.ConnectionState = f.newConnectionState(f.l, true, noise.HandshakeIX, []byte{}, 0, certState)
	}
}

func (f *Interface) sendMessageNow(t header.MessageType, st header.MessageSubType, hostInfo *HostInfo, p, nb, out []byte) {
	fp := &firewall.Packet{}
	err := newPacket(p, false, fp)
	if err != nil {
		f.l.Warnf("error while parsing outgoing packet for firewall check; %v", err)
		return
	}

	// check if packet is in outbound fw rules
	if f.caPool[hostInfo.networkID] != nil {
		dropReason := f.firewall.Drop(p, *fp, false, hostInfo, f.caPool[hostInfo.networkID], nil)
		if dropReason != nil {
			if f.l.Level >= logrus.DebugLevel {
				f.l.WithField("fwPacket", fp).
					WithField("reason", dropReason).
					Debugln("dropping cached packet")
			}
			return
		}
	}

	f.sendNoMetrics(header.Message, st, hostInfo.ConnectionState, hostInfo, hostInfo.remote, p, nb, out, 0)
}

// SendMessageToVpnIp handles real ip:port lookup and sends to the current best known address for vpnIp
func (f *Interface) SendMessageToVpnIp(t header.MessageType, st header.MessageSubType, vpnIp iputil.VpnIp, p, nb, out []byte, networkID uint64) {
	if isMulticast(vpnIp) {
		return
	}
	if vpnIp == 0 {
		return
	}
	hostInfo := f.getOrHandshake(vpnIp, networkID, true)
	if hostInfo == nil {
		if f.l.Level >= logrus.DebugLevel {
			f.l.WithField("vpnIp", vpnIp).
				Debugln("dropping SendMessageToVpnIp, vpnIp not in our CIDR or in unsafe routes")
		}
		return
	}

	if hostInfo.ConnectionState != nil && !hostInfo.ConnectionState.ready {
		// Because we might be sending stored packets, lock here to stop new things going to
		// the packet queue.
		hostInfo.ConnectionState.queueLock.Lock()
		if !hostInfo.ConnectionState.ready {
			hostInfo.cachePacket(f.l, t, st, p, f.sendMessageToVpnIp, f.cachedPacketMetrics)
			hostInfo.ConnectionState.queueLock.Unlock()
			return
		}
		hostInfo.ConnectionState.queueLock.Unlock()
	}

	f.sendMessageToVpnIp(t, st, hostInfo, p, nb, out)
	return
}

func (f *Interface) sendMessageToVpnIp(t header.MessageType, st header.MessageSubType, hostInfo *HostInfo, p, nb, out []byte) {
	f.send(t, st, hostInfo.ConnectionState, hostInfo, hostInfo.remote, p, nb, out)
}

func (f *Interface) send(t header.MessageType, st header.MessageSubType, ci *ConnectionState, hostinfo *HostInfo, remote *udp.Addr, p, nb, out []byte) {
	f.messageMetrics.Tx(t, st, 1)
	f.sendNoMetrics(t, st, ci, hostinfo, remote, p, nb, out, 0)
}

func (f *Interface) sendNoMetrics(t header.MessageType, st header.MessageSubType, ci *ConnectionState, hostinfo *HostInfo, remote *udp.Addr, p, nb, out []byte, q int) {
	if ci == nil {
		return
	}
	if ci.eKey == nil {
		//TODO: log warning
		return
	}

	var err error
	//TODO: enable if we do more than 1 tun queue
	//ci.writeLock.Lock()
	c := atomic.AddUint64(&ci.atomicMessageCounter, 1)

	out = header.Encode(out, header.Version, t, st, hostinfo.remoteIndexId, c, 0, 0, 0, 0, hostinfo.networkID)
	f.connectionManager.Out(hostinfo.vpnIp, hostinfo.networkID)

	// Query our LH if we haven't since the last time we've been rebound, this will cause the remote to punch against
	// all our IPs and enable a faster roaming.
	if t != header.CloseTunnel && hostinfo.lastRebindCount != f.rebindCount {
		//NOTE: there is an update hole if a tunnel isn't used and exactly 256 rebinds occur before the tunnel is
		// finally used again. This tunnel would eventually be torn down and recreated if this action didn't help.
		f.lightHouse.QueryServer(hostinfo.vpnIp, hostinfo.networkID, f)
		hostinfo.lastRebindCount = f.rebindCount
		if f.l.Level >= logrus.DebugLevel {
			f.l.WithField("vpnIp", hostinfo.vpnIp).Debug("Lighthouse update triggered for punch due to rebind counter")
		}
	}

	out, err = ci.eKey.EncryptDanger(out, out, p, c, nb)
	//TODO: see above note on lock
	//ci.writeLock.Unlock()
	if err != nil {
		hostinfo.logger(f.l).WithError(err).
			WithField("udpAddr", remote).WithField("counter", c).
			WithField("attemptedCounter", c).
			Error("Failed to encrypt outgoing packet")
		return
	}

	if hostinfo.relay == 1 {
		if t == header.CloseTunnel {
			f.l.WithField("Addr ", remote).Info("Closing tunnel")
		}
		if err == nil && f.certState != nil {
			f.SendRelay(header.RelayPacket, 0, out, make([]byte, 12, 12),
				make([]byte, mtu), (uint32)(hostinfo.vpnIp), udp.Ip2int(f.certState.certificate.Details.Ips[0].IP), 0, 0, hostinfo.networkID, hostinfo.relayIP)
			if t == header.CloseTunnel {
				f.l.WithField("Addr ", remote).Info("Closed tunnel")
			}
		}
	} else {
		err = f.writers[q].WriteTo(out, remote)
		if err != nil {
			hostinfo.logger(f.l).WithError(err).
				WithField("udpAddr", remote).Error("Failed to write outgoing packet")
		}
	}
	hostinfo.out_bytes += (uint64)(len(p))
	return
}

func (f *Interface) SendRelay(t header.MessageType, st header.MessageSubType, p, nb, out []byte, destIP uint32, sourceIP uint32, destPort uint16, sourcePort uint16, networkID uint64, relayIP *iputil.VpnIp) error {
	var hostinfoout *HostInfo
	if f.lightHouse.amLighthouse {
		// Look for hostinfo. But don't initiate handshake
		hostinfoout = f.getOrHandshake((iputil.VpnIp)(destIP), networkID, false)
	} else {
		// Check whetehr we have any desired relay server. As a first step get hostinfo of destination

		hostinfoout = nil
		if relayIP != nil {
			hostinfoout = f.getOrHandshake(*relayIP, networkID, false)
		} else {
			hostinfo := f.getOrHandshake((iputil.VpnIp)(destIP), networkID, false)
			if hostinfo != nil && hostinfo.relayIP != nil {
				// We have a preferred relay IP for this destination. Try it
				hostinfoout = f.getOrHandshake(*(hostinfo.relayIP), networkID, false)
			}
		}
		if hostinfoout == nil || hostinfoout.remote == nil || hostinfoout.ConnectionState == nil {
			// No preferred relay server. Try the best one that we know
			if f.relayHostInfo == nil {
				f.UpdateRelayHostInfo()
			}
			hostinfoout = f.relayHostInfo
		}
	}
	if hostinfoout == nil {
		// No need to log any error
		return fmt.Errorf("Relayhostinfo is nil")
	}
	ci := hostinfoout.ConnectionState

	remote := hostinfoout.remote
	if remote == nil {
		// No need to log any error
		return fmt.Errorf("remotes from Relayhostinfo is nil")
	}
	var err error
	//TODO: enable if we do more than 1 tun queue
	//ci.writeLock.Lock()
	c := atomic.AddUint64(&ci.atomicMessageCounter, 1)

	out = header.Encode(out, header.Version, (t), (st), hostinfoout.remoteIndexId, c, destIP, sourceIP, destPort, sourcePort, hostinfoout.networkID)
	f.connectionManager.Out(hostinfoout.vpnIp, hostinfoout.networkID)

	out = append(out, p...)
	//out, err = ci.eKey.EncryptDanger(out, out, p, c, nb)
	//TODO: see above note on lock
	//ci.writeLock.Unlock()
	if err != nil {
		hostinfoout.logger(f.l).WithError(err).
			WithField("udpAddr", remote).WithField("counter", c).
			WithField("attemptedCounter", ci.atomicMessageCounter).
			Error("Failed to encrypt outgoing packet")
		return fmt.Errorf("Failed to encrypt outgoing packet")
	}

	err = f.writers[0].WriteTo(out, remote)
	if err != nil {
		hostinfoout.logger(f.l).WithError(err).
			WithField("udpAddr", remote).Error("1. Failed to write outgoing packet")
	}
	return err
}

func (f *Interface) UpdateRelayHostInfo() {
	f.hostMap.Lock()
	defer f.hostMap.Unlock()
	if f.lightHouse.amLighthouse {
		return
	}
	ips := f.lightHouse.getLightHouseIPs()
	if ips == nil {
		return
	}
	var hostInfo *HostInfo
	var relayHostInfo *HostInfo

	relayHostInfo = nil
	for _, ip := range ips {
		if ip.String() == "0.0.0.0" {
			continue
		}
		hostInfo = f.hostMap.Hosts[f.networkID][ip]
		if hostInfo != nil && hostInfo.ConnectionState != nil &&
			hostInfo.ConnectionState.ready {
			if relayHostInfo == nil {
				relayHostInfo = hostInfo
			} else if relayHostInfo.hsDuration > hostInfo.hsDuration {
				relayHostInfo = hostInfo
			}
		}
	}
	if relayHostInfo == nil {
		return
	}
	if f.relayHostInfo != nil && f.relayHostInfo == relayHostInfo {
		// No change in relay server
		return
	}
	if f.relayHostInfo == nil || f.relayHostInfo.hsDuration > (relayHostInfo.hsDuration+50) {
		f.relayHostInfo = relayHostInfo
	}
}

func (f *Interface) amIConnectedWithThisIP(ip iputil.VpnIp) bool {
	f.hostMap.RLock()
	defer f.hostMap.RUnlock()
	hostInfo := f.hostMap.Hosts[f.networkID][ip]
	if hostInfo == nil {
		return false
	} else {
		return true
	}
}

func (f *Interface) GetABetterRelayServer() (ip *iputil.VpnIp) {
	f.hostMap.Lock()
	defer f.hostMap.Unlock()
	if f.lightHouse.amLighthouse {
		return
	}
	ips := f.lightHouse.getLightHouseIPs()
	if ips == nil {
		return
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	//rand.Shuffle(ips)
	var hostInfo *HostInfo

	for _, ip := range ips {
		if ip.String() == "0.0.0.0" {
			continue
		}
		hostInfo = f.hostMap.Hosts[f.networkID][ip]
		if hostInfo != nil && hostInfo.ConnectionState != nil &&
			hostInfo.ConnectionState.ready && f.relayHostInfo != hostInfo {
			return &(hostInfo.vpnIp)
		}
	}
	return nil
}

func isMulticast(ip iputil.VpnIp) bool {
	// Class D multicast
	if (((ip >> 24) & 0xff) & 0xf0) == 0xe0 {
		return true
	}

	return false
}
