package overlay

import (
	"fmt"
	"net"

	platform "platform"

	"github.com/jackpal/gateway"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/util"
)

const DefaultMTU = 1300

var full_vpn bool
var defaultIndex int
var localip net.IP
var defaultifname string
var defaultgwip net.IP

func getDefaultInterfaceIndex() (int, net.IP) {
	localip, err := gateway.DiscoverInterface()
	if err != nil {
		return -1, nil
	}
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				//continue
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.String() == localip.String() {
				return i.Index, localip
			}
		}
	}
	return -1, localip
}

func NewDeviceFromConfig(c *config.C, l *logrus.Logger, tunCidr *net.IPNet, fd *int, routines int) (Device, error) {
	routes, err := parseRoutes(c, tunCidr)
	if err != nil {
		return nil, util.NewContextualError("Could not parse tun.routes", nil, err)
	}
	l.Error("2. routes length...", len(routes))

	full_vpn = c.GetBool("tun.fullvpn", false)
	mobilevpn := c.GetBool("tun.mobilevpn", false)
	dns, err := getDNSServers(c)
	if err != nil {
		return nil, util.NewContextualError("Could not parse tun.dns", nil, err)
	}

	if full_vpn && !mobilevpn {
		defaultIndex, localip = getDefaultInterfaceIndex()
		// Get Default Gateway IP and default interface name
		defaultgwip, err = gateway.DiscoverGateway()
		if err != nil {
			return nil, util.NewContextualError("Could not get default gateway", nil, err)
		}

		defaultifname, err = platform.GetDefaultInterfaceName()
		if err != nil {
			return nil, util.NewContextualError("Error while getting interface name", nil, err)
		}
		// Neither interface index nor name available
		if defaultIndex < 0 && defaultifname == "" {
			return nil, util.NewContextualError("Could not get default local Interface. Check your Internet Connection", nil, fmt.Errorf("Check your Internet connection"))
		}
		relayRoutes, err := getRelayServerRoutes(c)
		if err != nil {
			return nil, util.NewContextualError("Could not parse Relay server ips", nil, err)
		}
		routes = append(routes, relayRoutes...)
		l.Error("Added Relay routes", len(routes))
	}
	l.Error("3. routes length...", len(routes))
	unsafeRoutes, err := parseUnsafeRoutes(c, tunCidr)
	if err != nil {
		return nil, util.NewContextualError("Could not parse tun.unsafe_routes", nil, err)
	}
	routes = append(routes, unsafeRoutes...)
	l.Error("1. routes length...", len(routes))

	switch {
	case c.GetBool("tun.disabled", false):
		tun := newDisabledTun(tunCidr, c.GetInt("tun.tx_queue", 500), c.GetBool("stats.message_metrics", false), l)
		return tun, nil

	case fd != nil:
		return newTunFromFd(
			l,
			*fd,
			tunCidr,
			c.GetInt("tun.mtu", DefaultMTU),
			routes,
			c.GetInt("tun.tx_queue", 500),
			dns,
		)

	default:
		return newTun(
			l,
			c.GetString("tun.dev", ""),
			tunCidr,
			c.GetInt("tun.mtu", DefaultMTU),
			routes,
			c.GetInt("tun.tx_queue", 500),
			routines > 1,
			dns,
		)
	}
}
