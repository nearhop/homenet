package udp

import (
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
)

type EncWriter interface {
	SendMessageToVpnIp(t header.MessageType, st header.MessageSubType, vpnIp iputil.VpnIp, p, nb, out []byte, networkID uint64)
	SendRelay(t header.MessageType, st header.MessageSubType, p, nb, out []byte, destIP uint32, sourceIP uint32, destPort uint16, sourcePort uint16, networkID uint64, relayIP *iputil.VpnIp) error
	GetABetterRelayServer() (ip *iputil.VpnIp)
}
type LightHouseHandlerFunc func(rAddr *Addr, vpnIp iputil.VpnIp, networkID uint64, p []byte, w EncWriter)
