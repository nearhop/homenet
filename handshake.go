package nebula

import (
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

func HandleIncomingHandshake(f *Interface, addr *udp.Addr, packet []byte, h *header.H, hostinfo *HostInfo, relay uint8, relayIP *iputil.VpnIp) {
	// First remote allow list check before we know the vpnIp
	if !f.lightHouse.remoteAllowList.AllowUnknownVpnIp(addr.IP) {
		f.l.WithField("udpAddr", addr).Debug("lighthouse.remote_allow_list denied incoming handshake")
		return
	}

	switch h.Subtype {
	case header.HandshakeIXPSK0:
		switch h.MessageCounter {
		case 1:
			ixHandshakeStage1(f, addr, packet, h, relay, relayIP)
		case 2:
			networkID := h.NetworkID
			if !f.lightHouse.amLighthouse {
				networkID = f.networkID
			}
			f.l.WithField("udpAddr", addr).WithField("networkID", networkID).Error("Debuggggggggggggggggggging")
			newHostinfo, _ := f.handshakeManager.QueryIndex(h.RemoteIndex, networkID)
			tearDown := ixHandshakeStage2(f, addr, newHostinfo, packet, h)
			if tearDown && newHostinfo != nil {
				f.handshakeManager.DeleteHostInfo(newHostinfo)
			}
		}
	}

}
