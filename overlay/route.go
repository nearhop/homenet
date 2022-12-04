package overlay

import (
	"fmt"
	"math"
	"net"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cidr"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/udp"
)

type Route struct {
	MTU     int
	Metric  int
	Cidr    *net.IPNet
	Via     *iputil.VpnIp
	Iname   string
	GW      net.IP
	Index   int
	LocalIP net.IP
	Name    string
}

func makeRouteTree(l *logrus.Logger, routes []Route, allowMTU bool) (*cidr.Tree4, error) {
	routeTree := cidr.NewTree4()
	for _, r := range routes {
		if !allowMTU && r.MTU > 0 {
			l.WithField("route", r).Warnf("route MTU is not supported in %s", runtime.GOOS)
		}

		if r.Via != nil {
			routeTree.AddCIDR(r.Cidr, *r.Via)
		}
	}
	return routeTree, nil
}

func parseRoutes(c *config.C, network *net.IPNet) ([]Route, error) {
	var err error

	r := c.Get("tun.routes")
	if r == nil {
		return []Route{}, nil
	}

	rawRoutes, ok := r.([]interface{})
	if !ok {
		return nil, fmt.Errorf("tun.routes is not an array")
	}

	if len(rawRoutes) < 1 {
		return []Route{}, nil
	}

	routes := make([]Route, len(rawRoutes))
	for i, r := range rawRoutes {
		m, ok := r.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("entry %v in tun.routes is invalid", i+1)
		}
		rMtu, ok := m["mtu"]
		if !ok {
			return nil, fmt.Errorf("entry %v.mtu in tun.routes is not present", i+1)
		}

		mtu, ok := rMtu.(int)
		if !ok {
			mtu, err = strconv.Atoi(rMtu.(string))
			if err != nil {
				return nil, fmt.Errorf("entry %v.mtu in tun.routes is not an integer: %v", i+1, err)
			}
		}

		if mtu < 500 {
			return nil, fmt.Errorf("entry %v.mtu in tun.routes is below 500: %v", i+1, mtu)
		}

		rRoute, ok := m["route"]
		if !ok {
			return nil, fmt.Errorf("entry %v.route in tun.routes is not present", i+1)
		}

		r := Route{
			MTU: mtu,
		}

		_, r.Cidr, err = net.ParseCIDR(fmt.Sprintf("%v", rRoute))
		if err != nil {
			return nil, fmt.Errorf("entry %v.route in tun.routes failed to parse: %v", i+1, err)
		}

		if !ipWithin(network, r.Cidr) {
			return nil, fmt.Errorf(
				"entry %v.route in tun.routes is not contained within the network attached to the certificate; route: %v, network: %v",
				i+1,
				r.Cidr.String(),
				network.String(),
			)
		}

		routes[i] = r
	}

	return routes, nil
}

func parseUnsafeRoutes(c *config.C, network *net.IPNet) ([]Route, error) {
	var err error

	r := c.Get("tun.unsafe_routes")
	if r == nil {
		return []Route{}, nil
	}

	rawRoutes, ok := r.([]interface{})
	if !ok {
		return nil, fmt.Errorf("tun.unsafe_routes is not an array")
	}

	if len(rawRoutes) < 1 {
		return []Route{}, nil
	}

	routes := make([]Route, len(rawRoutes))
	for i, r := range rawRoutes {
		m, ok := r.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("entry %v in tun.unsafe_routes is invalid", i+1)
		}

		var mtu int
		if rMtu, ok := m["mtu"]; ok {
			mtu, ok = rMtu.(int)
			if !ok {
				mtu, err = strconv.Atoi(rMtu.(string))
				if err != nil {
					return nil, fmt.Errorf("entry %v.mtu in tun.unsafe_routes is not an integer: %v", i+1, err)
				}
			}

			if mtu != 0 && mtu < 500 {
				return nil, fmt.Errorf("entry %v.mtu in tun.unsafe_routes is below 500: %v", i+1, mtu)
			}
		}

		rMetric, ok := m["metric"]
		if !ok {
			rMetric = 0
		}

		metric, ok := rMetric.(int)
		if !ok {
			_, err = strconv.ParseInt(rMetric.(string), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("entry %v.metric in tun.unsafe_routes is not an integer: %v", i+1, err)
			}
		}

		if metric < 0 || metric > math.MaxInt32 {
			return nil, fmt.Errorf("entry %v.metric in tun.unsafe_routes is not in range (0-%d) : %v", i+1, math.MaxInt32, metric)
		}

		rName, ok := m["name"]
		var name string

		name = ""
		if ok {
			name, ok = rName.(string)
			if !ok {
				name = ""
			}
		}

		rVia, ok := m["via"]
		if !ok {
			return nil, fmt.Errorf("entry %v.via in tun.unsafe_routes is not present", i+1)
		}

		via, ok := rVia.(string)
		if !ok {
			return nil, fmt.Errorf("entry %v.via in tun.unsafe_routes is not a string: found %T", i+1, rVia)
		}

		nVia := net.ParseIP(via)
		if nVia == nil {
			return nil, fmt.Errorf("entry %v.via in tun.unsafe_routes failed to parse address: %v", i+1, via)
		}

		rRoute, ok := m["route"]
		if !ok {
			return nil, fmt.Errorf("entry %v.route in tun.unsafe_routes is not present", i+1)
		}

		viaVpnIp := iputil.Ip2VpnIp(nVia)

		r := Route{
			Via:    &viaVpnIp,
			MTU:    mtu,
			Metric: metric,
			Name:   name,
		}

		_, r.Cidr, err = net.ParseCIDR(fmt.Sprintf("%v", rRoute))
		if err != nil {
			return nil, fmt.Errorf("entry %v.route in tun.unsafe_routes failed to parse: %v", i+1, err)
		}

		if network != nil && ipWithin(network, r.Cidr) {
			return nil, fmt.Errorf(
				"entry %v.route in tun.unsafe_routes is contained within the network attached to the certificate; route: %v, network: %v",
				i+1,
				r.Cidr.String(),
				network.String(),
			)
		}

		routes[i] = r
	}

	return routes, nil
}

func ParseUnsafeRoutes(c *config.C, network *net.IPNet) ([]Route, error) {
	return parseUnsafeRoutes(c, network)
}

func CreateRouteEntry(destip string, gwip net.IP, ifname string, defaultIndex int, localip net.IP) (*Route, error) {
	nVia := net.ParseIP(gwip.String())
	if nVia == nil {
		return nil, fmt.Errorf("Failed to parse server address: %v", gwip)
	}

	r := &Route{
		GW:      nVia,
		MTU:     1100,
		Metric:  0,
		Iname:   ifname,
		Index:   defaultIndex,
		LocalIP: localip,
	}

	var err error
	_, r.Cidr, err = net.ParseCIDR(destip)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ip :", destip)
	}
	return r, nil
}

func getRelayServerRoutes(c *config.C) ([]Route, error) {
	var servers [32]string
	num_of_servers := 0
	for _, v := range c.GetMap("static_host_map", map[interface{}]interface{}{}) {
		vals, ok := v.([]interface{})
		if ok {
			for _, v := range vals {
				ip, _, err := udp.ParseIPAndPort(fmt.Sprintf("%v", v))
				servers[num_of_servers] = ip.String()
				num_of_servers++
				if err != nil {
					return nil, fmt.Errorf("Error while parsing relayserver entries from config")
				}
			}
		}
	}
	routes := make([]Route, num_of_servers)
	for i := 0; i < num_of_servers; i++ {
		serverip := servers[i] + "/32"
		r, err := CreateRouteEntry(serverip, defaultgwip, defaultifname, defaultIndex, localip)
		if err != nil {
			return nil, err
		}
		routes[i] = *r
	}
	return routes, nil
}

func getDNSServers(c *config.C) ([]net.IP, error) {
	// tun.dns value (if exists) is a comma separated string of dns servers
	dnsString := c.GetString("tun.dns", "")
	if dnsString == "" {
		return nil, nil
	}
	dnsServers := strings.Split(dnsString, ",")
	servers := make([]net.IP, len(dnsServers))
	if servers == nil {
		return nil, fmt.Errorf("Error while allocating memory for tun dns serverss")
	}
	for i, server := range dnsServers {
		servers[i] = net.ParseIP(strings.Trim(server, " "))
	}
	return servers, nil
}

func GetDNSServers(c *config.C) ([]net.IP, error) {
	return getDNSServers(c)
}

func AddRoutes(ips []net.IP, t Device) error {
	if !full_vpn {
		return nil
	}
	routes := make([]Route, len(ips))
	for i, ip := range ips {
		r, err := CreateRouteEntry(ip.String()+"/32", defaultgwip, defaultifname, defaultIndex, localip)
		if err != nil {
			return err
		}
		routes[i] = *r
	}
	return t.AddRoutes(routes)
}

func ipWithin(o *net.IPNet, i *net.IPNet) bool {
	// Make sure o contains the lowest form of i
	if !o.Contains(i.IP.Mask(i.Mask)) {
		return false
	}

	// Find the max ip in i
	ip4 := i.IP.To4()
	if ip4 == nil {
		return false
	}

	last := make(net.IP, len(ip4))
	copy(last, ip4)
	for x := range ip4 {
		last[x] |= ^i.Mask[x]
	}

	// Make sure o contains the max
	if !o.Contains(last) {
		return false
	}

	return true
}
