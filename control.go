package nebula

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

// Every interaction here needs to take extra care to copy memory and not return or use arguments "as is" when touching
// core. This means copying IP objects, slices, de-referencing pointers and taking the actual value, etc

type Control struct {
	f          *Interface
	l          *logrus.Logger
	cancel     context.CancelFunc
	sshStart   func()
	statsStart func()
	dnsStart   func()
	appMsg     MobileNetCallBack
}

type ControlHostInfo struct {
	VpnIp          net.IP                  `json:"vpnIp"`
	LocalIndex     uint32                  `json:"localIndex"`
	RemoteIndex    uint32                  `json:"remoteIndex"`
	RemoteAddrs    []*udp.Addr             `json:"remoteAddrs"`
	CachedPackets  int                     `json:"cachedPackets"`
	Cert           *cert.NebulaCertificate `json:"cert"`
	MessageCounter uint64                  `json:"messageCounter"`
	CurrentRemote  *udp.Addr               `json:"currentRemote"`
	Relay          uint8                   `json:"relay"`
	In_bytes       uint64                  `json:"in_bytes"`
	Out_bytes      uint64                  `json:"out_bytes"`
	Name           string                  `json:"name"`
	VpnMode        uint8                   `json:"vpnmode"`
}

// Start actually runs nebula, this is a nonblocking call. To block use Control.ShutdownBlock()
func (c *Control) Start() {
	// Activate the interface
	c.f.activate()

	// Call all the delayed funcs that waited patiently for the interface to be created.
	if c.sshStart != nil {
		go c.sshStart()
	}
	if c.statsStart != nil {
		go c.statsStart()
	}
	if c.dnsStart != nil {
		go c.dnsStart()
	}

	// Start reading packets.
	c.f.run()
}

// Stop signals nebula to shutdown, returns after the shutdown is complete
func (c *Control) Stop() {
	//TODO: stop tun and udp routines, the lock on hostMap effectively does that though
	c.CloseAllTunnels(true)
	c.CloseAllTunnels(false)
	if err := c.f.Close(); err != nil {
		c.l.WithError(err).Error("Close interface failed")
	}
	c.cancel()
	c.l.Info("Goodbye")
}

// ShutdownBlock will listen for and block on term and interrupt signals, calling Control.Stop() once signalled
func (c *Control) ShutdownBlock() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)

	rawSig := <-sigChan
	sig := rawSig.String()
	c.l.WithField("signal", sig).Info("Caught signal, shutting down")
	c.Stop()
}

// RebindUDPServer asks the UDP listener to rebind it's listener. Mainly used on mobile clients when interfaces change
func (c *Control) RebindUDPServer() {
	_ = c.f.outside.Rebind()

	// Trigger a lighthouse update, useful for mobile clients that should have an update interval of 0
	c.f.lightHouse.SendUpdate(c.f)

	// Let the main interface know that we rebound so that underlying tunnels know to trigger punches from their remotes
	c.f.rebindCount++
}

// ListHostmap returns details about the actual or pending (handshaking) hostmap
func (c *Control) ListHostmap(pendingMap bool) []ControlHostInfo {
	if pendingMap {
		return listHostMap(c.f.handshakeManager.pendingHostMap, c.f.networkID)
	} else {
		return listHostMap(c.f.hostMap, c.f.networkID)
	}
}

type MobileNetCallBack interface {
	SendMessageToMobile(string)
}

func (c *Control) RegisterAppCallBack(cb MobileNetCallBack) {
	fmt.Printf("Nebula RegisterMobileAppCallBack called")
	if c.appMsg == nil {
		c.appMsg = cb
	}
	fmt.Printf("Nebula RegisterMobileAppCallBack calling SendMessageToMobile")
	c.appMsg.SendMessageToMobile("RegisterAppCallBack")
}

func (c *Control) SendMessageToMobile(message string) {
	if c.appMsg != nil {
		c.appMsg.SendMessageToMobile(message)
	}
}

func (c *Control) SendNonTunMessage(vpnIp iputil.VpnIp, message []byte) (string, error) {
	return c.f.messaging.sendMessage(vpnIp, c.f.networkID, message, header.NonTunMessageMain, 0)
}

func (c *Control) GetNextEvent() string {
	return c.f.messaging.GetNextEvent()
}

// GetHostInfoByVpnIp returns a single tunnels hostInfo, or nil if not found
func (c *Control) GetRouterPublicIP(vpnIp iputil.VpnIp, pending bool) *ControlHostInfo {
	var hm *HostMap
	if pending {
		hm = c.f.handshakeManager.pendingHostMap
	} else {
		hm = c.f.hostMap
	}

	h, err := hm.QueryVpnIp(vpnIp, c.f.networkID)
	if err != nil {
		return nil
	}

	ch := copyHostInfo(h, c.f.hostMap.preferredRanges)
	return &ch
}

func (c *Control) GetHostInfoByVpnIp(vpnIp iputil.VpnIp, pending bool) *ControlHostInfo {
	var hm *HostMap
	if pending {
		hm = c.f.handshakeManager.pendingHostMap
	} else {
		hm = c.f.hostMap
	}

	h, err := hm.QueryVpnIp(vpnIp, c.f.networkID)
	if err != nil {
		return nil
	}

	ch := copyHostInfo(h, c.f.hostMap.preferredRanges)
	return &ch
}

// SetRemoteForTunnel forces a tunnel to use a specific remote
func (c *Control) SetRemoteForTunnel(vpnIp iputil.VpnIp, addr udp.Addr) *ControlHostInfo {
	hostInfo, err := c.f.hostMap.QueryVpnIp(vpnIp, c.f.networkID)
	if err != nil {
		return nil
	}

	hostInfo.SetRemote(addr.Copy())
	ch := copyHostInfo(hostInfo, c.f.hostMap.preferredRanges)
	return &ch
}

// CloseTunnel closes a fully established tunnel. If localOnly is false it will notify the remote end as well.
func (c *Control) CloseTunnel(vpnIp iputil.VpnIp, localOnly bool) bool {
	hostInfo, err := c.f.hostMap.QueryVpnIp(vpnIp, c.f.networkID)
	if err != nil {
		return false
	}

	if !localOnly {
		c.f.send(
			header.CloseTunnel,
			0,
			hostInfo.ConnectionState,
			hostInfo,
			hostInfo.remote,
			[]byte{},
			make([]byte, 12, 12),
			make([]byte, mtu),
		)
	}

	c.f.closeTunnel(hostInfo, false, c.f.networkID)
	return true
}

// CloseAllTunnels is just like CloseTunnel except it goes through and shuts them all down, optionally you can avoid shutting down lighthouse tunnels
// the int returned is a count of tunnels closed
func (c *Control) CloseAllTunnels(excludeLighthouses bool) (closed int) {
	//TODO: this is probably better as a function in ConnectionManager or HostMap directly
	//c.f.hostMap.Lock()
	for _, hosts1 := range c.f.hostMap.Hosts {
		for _, h := range hosts1 {
			if excludeLighthouses {
				if _, ok := c.f.lightHouse.lighthouses[h.vpnIp]; ok {
					continue
				}
			}

			if h.ConnectionState.ready {
				c.f.send(header.CloseTunnel, 0, h.ConnectionState, h, h.remote, []byte{}, make([]byte, 12, 12), make([]byte, mtu))
				c.f.closeTunnel(h, true, c.f.networkID)

				c.l.WithField("vpnIp", h.vpnIp).WithField("udpAddr", h.remote).
					Info("Sending close tunnel message")
				closed++
			}
		}
	}
	//c.f.hostMap.Unlock()
	return
}

func copyHostInfo(h *HostInfo, preferredRanges []*net.IPNet) ControlHostInfo {

	chi := ControlHostInfo{
		VpnIp:         h.vpnIp.ToIP(),
		Relay:         h.relay,
		LocalIndex:    h.localIndexId,
		RemoteIndex:   h.remoteIndexId,
		RemoteAddrs:   h.remotes.CopyAddrs(preferredRanges),
		CachedPackets: len(h.packetStore),
		In_bytes:      h.in_bytes,
		Out_bytes:     h.out_bytes,
		Name:          h.name,
		VpnMode:       h.VpnMode,
	}

	if h.ConnectionState != nil {
		chi.MessageCounter = atomic.LoadUint64(&h.ConnectionState.atomicMessageCounter)
	}

	if c := h.GetCert(); c != nil {
		chi.Cert = c.Copy()
	}

	if h.remote != nil {
		chi.CurrentRemote = h.remote.Copy()
	}

	return chi
}

func listHostMap(hm *HostMap, networkID uint64) []ControlHostInfo {
	hm.RLock()
	hosts := make([]ControlHostInfo, len(hm.Hosts[networkID]))
	i := 0
	for _, v := range hm.Hosts[networkID] {
		hosts[i] = copyHostInfo(v, hm.preferredRanges)
		hm.l.Info(" VPN MODE  IS ", hosts[i].VpnMode, " FOR IP ", hosts[i].VpnIp)
		i++
	}
	hm.RUnlock()

	return hosts
}

func (c *Control) GetName() string {
	if c.f.lightHouse.amLighthouse {
		return "server"
	} else {
		return c.f.Name
	}
}

func (c *Control) GetMyVPNIP() string {
	if c.f.lightHouse.amLighthouse {
		return ""
	} else {
		return c.f.myVpnIp.String()
	}
}

func (c *Control) GetRelayHostIP() string {
	if c.f.relayHostInfo == nil {
		return ""
	} else {
		return c.f.relayHostInfo.vpnIp.String()
	}
}

func (c *Control) SendMessage(vpnIp iputil.VpnIp, message string) (string, error) {
	return c.f.messaging.sendMessage(vpnIp, c.f.networkID, []byte(message), header.NonTunMessageMain, 0)
}
