package nebula

import (
	nh_util "nh_util"

	"github.com/slackhq/nebula/firewall"
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

type RelayServer struct {
	relayVPNIP      iputil.VpnIp
	relayPublicAddr *udp.Addr
}

func NewRelayServer(relayIP iputil.VpnIp, relayAddr *udp.Addr) *RelayServer {
	rServer := RelayServer{
		relayVPNIP:      relayIP,
		relayPublicAddr: relayAddr,
	}
	return &rServer
}

func HandleRelay(f *Interface, addr *udp.Addr, out []byte, packet []byte, h *header.H, hostinfo *HostInfo, nb []byte, q int, localCache firewall.ConntrackCache) {
	//d, err := f.decrypt(hostinfo, header.MessageCounter, out, packet, header, nb)
	d := packet[header.Len:]
	//f.decryptToTun(hostinfo, header.MessageCounter, out, packet, fwPacket, nb, q, localCache)
	//localIP := binary.BigEndian.Uint32(out[12:16])
	//f.l.WithField("sourceIP", nh_util.Int2ip(h.SourceIP)).WithField("destIP", nh_util.Int2ip(h.DestIP)).Info("Relay packet: IP ")
	if f.lightHouse.amLighthouse {
		dhostinfo := f.getOrHandshake((iputil.VpnIp)(h.DestIP), h.NetworkID, false)
		if dhostinfo != nil {
			//f.l.WithField("sourceIP", nh_util.Int2ip(h.SourceIP)).WithField("destIP", nh_util.Int2ip(h.DestIP)).Info("Forwarding Relay packet: IP ")
			err := f.SendRelay(header.RelayPacket, 0, d, make([]byte, 12, 12), make([]byte, mtu), h.DestIP, h.SourceIP, h.DestPort, h.SourcePort, h.NetworkID, nil)
			f.l.WithField("sourceIP", nh_util.Int2ip(h.SourceIP)).WithField("destIP", nh_util.Int2ip(h.DestIP)).Info("Relay packet: IP ", err)

		}
	} else {
		headerNew := &header.H{}
		headerNew.Parse(d)
		hostinfoNew, err := f.hostMap.QueryIndex(headerNew.RemoteIndex, headerNew.NetworkID)
		addrNew := udp.NewAddr(udp.Int2ip(h.SourceIP), h.SourcePort)

		var ci *ConnectionState
		if err == nil {
			ci = hostinfoNew.ConnectionState
		}
		switch headerNew.Type {
		case header.Message:
			fwPacketNew := &firewall.Packet{}
			outNew := make([]byte, mtu)
			nbNew := make([]byte, 12, 12)
			if !f.handleEncrypted(ci, addrNew, headerNew) {
				return
			}
			hostinfoNew.in_bytes += (uint64)(len(packet))
			f.decryptToTun(hostinfoNew, headerNew.MessageCounter, outNew[:0], d, fwPacketNew, nbNew, q, localCache)
			return
		case header.Handshake:
			hostinfo.logger(f.l).WithField("addrNewwww", addrNew).WithField("RelayIP", hostinfo.vpnIp).Error("Received handshake ")
			// Check if we are connected on the relay server used by the sender
			// Dont proceed otherwise
			if f.amIConnectedWithThisIP(hostinfo.vpnIp) {
				HandleIncomingHandshake(f, addrNew, d, headerNew, hostinfoNew, 1, &hostinfo.vpnIp)
			}
			return
		case header.CloseTunnel:
			if !f.handleEncrypted(ci, addrNew, headerNew) {
				return
			}

			hostinfo.logger(f.l).WithField("udpAddr", addr).
				Info("Close tunnel received, tearing down.")
			f.closeTunnel(hostinfoNew, false, f.networkID)
			return
		case header.NonTunMessage:
			nbNew := make([]byte, 12, 12)
			f.messageMetrics.Rx(headerNew.Type, headerNew.Subtype, 1)
			if !f.handleEncrypted(ci, addrNew, headerNew) {
				return
			}

			dec, err := f.decrypt(hostinfoNew, headerNew.MessageCounter, out, d, headerNew, nbNew)

			if err != nil {
				hostinfo.logger(f.l).WithError(err).WithField("udpAddr", addr).
					WithField("packet", len(packet)).
					Error("Failed to decrypt lighthouse packet")

				return
			}
			dcopy := make([]byte, len(dec))
			copy(dcopy, dec)
			//hostinfo.logger().Error("Nontun packet received from ", string(d))
			f.connectionManager.In(hostinfoNew.vpnIp)
			go f.messaging.recvMessage(hostinfoNew.vpnIp, dcopy)
			return
		default:
			hostinfo.logger(f.l).WithField("addr", addr).Error("2. Unexpected packet ", headerNew.Type)
			return
		}
	}
}
