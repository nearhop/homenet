package nebula

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	router "router"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/iputil"
	"github.com/slackhq/nebula/overlay"
	"github.com/slackhq/nebula/sshd"
	"github.com/slackhq/nebula/udp"
	"github.com/slackhq/nebula/util"
	"gopkg.in/yaml.v2"
)

type m map[string]interface{}

func Main(c *config.C, configTest bool, buildVersion string, logger *logrus.Logger, tunFd *int) (retcon *Control, rs *router.RouterServer, reterr error) {
	ctx, cancel := context.WithCancel(context.Background())
	// Automatically cancel the context if Main returns an error, to signal all created goroutines to quit.
	defer func() {
		if reterr != nil {
			cancel()
		}
	}()

	l := logger
	l.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}
	rs, err := router.NewRouterServer(l)
	if err == nil {
		go rs.StartRouterServer()
	} else {
		return nil, nil, err
	}
	// Print the config if in test, the exit comes later
	if configTest {
		b, err := yaml.Marshal(c.Settings)
		if err != nil {
			return nil, nil, err
		}

		// Print the final config
		l.Println(string(b))
	}

	err = configLogger(l, c)
	if err != nil {
		return nil, nil, util.NewContextualError("Failed to configure the logger", nil, err)
	}

	c.RegisterReloadCallback(func(c *config.C) {
		err := configLogger(l, c)
		if err != nil {
			l.WithError(err).Error("Failed to configure the logger")
		}
	})

	caFile, err := getCAFileFromConfig(c)
	if err != nil {
		//The errors coming out of loadCA are already nicely formatted
		return nil, nil, util.NewContextualError("Failed to get ca file from config", nil, err)
	}

	amLighthouse := c.GetBool("lighthouse.am_lighthouse", false)
	var cs *CertState
	var networkID uint64
	var tunCidr *net.IPNet
	var fw *Firewall
	var name string
	var relayIndex byte

	cs = nil
	networkID = 0
	tunCidr = nil
	fw = nil
	name = ""
	relayIndex = 0

	if !amLighthouse {
		cs, err = NewCertStateFromConfig(c)
		if err != nil {
			//The errors coming out of NewCertStateFromConfig are already nicely formatted
			return nil, nil, util.NewContextualError("Failed to load certificate from config", nil, err)
		}
		l.WithField("cert", cs.certificate).Debug("Client nebula certificate")
		networkID = cs.certificate.Details.NetworkID
		name = cs.certificate.Details.Name
		tunCidr = cs.certificate.Details.Ips[0]
		l.WithField("tunCidr", tunCidr).Info("tunCidrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr")

		fw, err = NewFirewallFromConfig(l, cs.certificate, c)
		if err != nil {
			return nil, nil, util.NewContextualError("Error while loading firewall rules", nil, err)
		}
		l.WithField("firewallHash", fw.GetRuleHash()).Info("Firewall started")
	} else {
		fmt.Println("Relay Index ...", c.GetInt("lighthouse.relay_index", 1))
		relayIndex = (byte)(c.GetInt("lighthouse.relay_index", 1))
		tunCidr = &net.IPNet{IP: net.IP{172, 16, 128, relayIndex}, Mask: net.IPMask{255, 255, 255, 0}}
		name = "Server" + string(relayIndex)
		l.WithField("network", tunCidr).
			Info("Nebula interface is activeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeiiiiiiiiiiiiiiiiii")
	}
	l.WithField("network", tunCidr).
		Info("Nebula interface is activeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeiiiiiiiiiiiiiiiiii")

	// TODO: make sure mask is 4 bytes
	ssh, err := sshd.NewSSHServer(l.WithField("subsystem", "sshd"))
	wireSSHReload(l, ssh, c)
	var sshStart func()
	if c.GetBool("sshd.enabled", false) {
		sshStart, err = configSSH(l, ssh, c)
		if err != nil {
			return nil, nil, util.NewContextualError("Error while configuring the sshd", nil, err)
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// All non system modifying configuration consumption should live above this line
	// tun config, listeners, anything modifying the computer should be below
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	var routines int

	// If `routines` is set, use that and ignore the specific values
	if routines = c.GetInt("routines", 0); routines != 0 {
		if routines < 1 {
			routines = 1
		}
		if routines > 1 {
			l.WithField("routines", routines).Info("Using multiple routines")
		}
	} else {
		// deprecated and undocumented
		tunQueues := c.GetInt("tun.routines", 1)
		udpQueues := c.GetInt("listen.routines", 1)
		if tunQueues > udpQueues {
			routines = tunQueues
		} else {
			routines = udpQueues
		}
		if routines != 1 {
			l.WithField("routines", routines).Warn("Setting tun.routines and listen.routines is deprecated. Use `routines` instead")
		}
	}

	// EXPERIMENTAL
	// Intentionally not documented yet while we do more testing and determine
	// a good default value.
	conntrackCacheTimeout := c.GetDuration("firewall.conntrack.routine_cache_timeout", 0)
	if routines > 1 && !c.IsSet("firewall.conntrack.routine_cache_timeout") {
		// Use a different default if we are running with multiple routines
		conntrackCacheTimeout = 1 * time.Second
	}
	if conntrackCacheTimeout > 0 {
		l.WithField("duration", conntrackCacheTimeout).Info("Using routine-local conntrack cache")
	}

	var tun overlay.Device
	if !configTest {
		c.CatchHUP(ctx)

		tun, err = overlay.NewDeviceFromConfig(c, l, tunCidr, tunFd, routines)
		if err != nil {
			return nil, nil, util.NewContextualError("Failed to get a tun/tap device", nil, err)
		}

		defer func() {
			if reterr != nil {
				tun.Close()
			}
		}()
	}

	// set up our UDP listener
	udpConns := make([]*udp.Conn, routines)
	port := c.GetInt("listen.port", 0)

	if !configTest {
		for i := 0; i < routines; i++ {
			udpServer, err := udp.NewListener(l, c.GetString("listen.host", "0.0.0.0"), port, routines > 1, c.GetInt("listen.batch", 64))
			if err != nil {
				return nil, nil, util.NewContextualError("Failed to open udp listener", m{"queue": i}, err)
			}
			udpServer.ReloadConfig(c)
			udpConns[i] = udpServer

			// If port is dynamic, discover it
			if port == 0 {
				uPort, err := udpServer.LocalAddr()
				if err != nil {
					return nil, nil, util.NewContextualError("Failed to get listening port", nil, err)
				}
				port = int(uPort.Port)
			}
		}
	}

	// Set up my internal host map
	var preferredRanges []*net.IPNet
	rawPreferredRanges := c.GetStringSlice("preferred_ranges", []string{})
	// First, check if 'preferred_ranges' is set and fallback to 'local_range'
	if len(rawPreferredRanges) > 0 {
		for _, rawPreferredRange := range rawPreferredRanges {
			_, preferredRange, err := net.ParseCIDR(rawPreferredRange)
			if err != nil {
				return nil, nil, util.NewContextualError("Failed to parse preferred ranges", nil, err)
			}
			preferredRanges = append(preferredRanges, preferredRange)
		}
	}

	// local_range was superseded by preferred_ranges. If it is still present,
	// merge the local_range setting into preferred_ranges. We will probably
	// deprecate local_range and remove in the future.
	rawLocalRange := c.GetString("local_range", "")
	if rawLocalRange != "" {
		_, localRange, err := net.ParseCIDR(rawLocalRange)
		if err != nil {
			return nil, nil, util.NewContextualError("Failed to parse local_range", nil, err)
		}

		// Check if the entry for local_range was already specified in
		// preferred_ranges. Don't put it into the slice twice if so.
		var found bool
		for _, r := range preferredRanges {
			if r.String() == localRange.String() {
				found = true
				break
			}
		}
		if !found {
			preferredRanges = append(preferredRanges, localRange)
		}
	}

	hostMap := NewHostMap(l, "main", tunCidr, preferredRanges)
	hostMap.metricsEnabled = c.GetBool("stats.message_metrics", false)

	l.WithField("network", hostMap.vpnCIDR).WithField("preferredRanges", hostMap.preferredRanges).Info("Main HostMap created")

	/*
		config.SetDefault("promoter.interval", 10)
		go hostMap.Promoter(config.GetInt("promoter.interval"))
	*/

	punchy := NewPunchyFromConfig(c)
	if punchy.Punch && !configTest {
		l.Info("UDP hole punching enabled")
		go hostMap.Punchy(ctx, udpConns[0])
	}

	// fatal if am_lighthouse is enabled but we are using an ephemeral port
	if amLighthouse && (c.GetInt("listen.port", 0) == 0) {
		return nil, nil, util.NewContextualError("lighthouse.am_lighthouse enabled on node but no port number is set in config", nil, nil)
	}

	// warn if am_lighthouse is enabled but upstream lighthouses exists
	rawLighthouseHosts := c.GetStringSlice("lighthouse.hosts", []string{})
	if amLighthouse && len(rawLighthouseHosts) != 0 {
		l.Warn("lighthouse.am_lighthouse enabled on node but upstream lighthouses exist in config")
	}

	lighthouseHosts := make([]iputil.VpnIp, len(rawLighthouseHosts))
	for i, host := range rawLighthouseHosts {
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, nil, util.NewContextualError("Unable to parse lighthouse host entry", m{"host": host, "entry": i + 1}, nil)
		}
		l.WithField("tunCidr", tunCidr).Info("1.....tunCidrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr")
		if !tunCidr.Contains(ip) {
			return nil, nil, util.NewContextualError("lighthouse host is not in our subnet, invalid", m{"vpnIp": ip, "network": tunCidr.String()}, nil)
		}
		lighthouseHosts[i] = iputil.Ip2VpnIp(ip)
	}

	if !amLighthouse && len(lighthouseHosts) == 0 {
		l.Warn("No lighthouses.hosts configured, this host will only be able to initiate tunnels with static_host_map entries")
	}

	lightHouse := NewLightHouse(
		l,
		amLighthouse,
		tunCidr,
		lighthouseHosts,
		//TODO: change to a duration
		c.GetInt("lighthouse.interval", 10),
		uint32(port),
		udpConns[0],
		punchy.Respond,
		punchy.Delay,
		c.GetBool("stats.lighthouse_metrics", false),
		networkID,
	)

	remoteAllowList, err := NewRemoteAllowListFromConfig(c, "lighthouse.remote_allow_list", "lighthouse.remote_allow_ranges")
	if err != nil {
		return nil, nil, util.NewContextualError("Invalid lighthouse.remote_allow_list", nil, err)
	}
	lightHouse.SetRemoteAllowList(remoteAllowList)

	localAllowList, err := NewLocalAllowListFromConfig(c, "lighthouse.local_allow_list")
	if err != nil {
		return nil, nil, util.NewContextualError("Invalid lighthouse.local_allow_list", nil, err)
	}
	lightHouse.SetLocalAllowList(localAllowList)

	var relayServer *RelayServer
	//TODO: Move all of this inside functions in lighthouse.go
	for k, v := range c.GetMap("static_host_map", map[interface{}]interface{}{}) {
		ip := net.ParseIP(fmt.Sprintf("%v", k))
		vpnIp := iputil.Ip2VpnIp(ip)
		if !tunCidr.Contains(ip) {
			return nil, nil, util.NewContextualError("static_host_map key is not in our subnet, invalid", m{"vpnIp": vpnIp, "network": tunCidr.String()}, nil)
		}
		vals, ok := v.([]interface{})
		if ok {
			for _, v := range vals {
				ip, port, err := udp.ParseIPAndPort(fmt.Sprintf("%v", v))
				if err != nil {
					return nil, nil, util.NewContextualError("Static host address could not be parsed", m{"vpnIp": vpnIp}, err)
				}
				lightHouse.AddStaticRemote(vpnIp, udp.NewAddr(ip, port), networkID)
				relayServer = NewRelayServer(vpnIp, udp.NewAddr(ip, port))
			}
		} else {
			ip, port, err := udp.ParseIPAndPort(fmt.Sprintf("%v", v))
			if err != nil {
				return nil, nil, util.NewContextualError("Static host address could not be parsed", m{"vpnIp": vpnIp}, err)
			}
			lightHouse.AddStaticRemote(vpnIp, udp.NewAddr(ip, port), networkID)
		}
	}

	err = lightHouse.ValidateLHStaticEntries()
	if err != nil {
		l.WithError(err).Error("Lighthouse unreachable")
	}

	var messageMetrics *MessageMetrics
	if c.GetBool("stats.message_metrics", false) {
		messageMetrics = newMessageMetrics()
	} else {
		messageMetrics = newMessageMetricsOnlyRecvError()
	}

	handshakeConfig := HandshakeConfig{
		tryInterval:   c.GetDuration("handshakes.try_interval", DefaultHandshakeTryInterval),
		retries:       c.GetInt("handshakes.retries", DefaultHandshakeRetries),
		triggerBuffer: c.GetInt("handshakes.trigger_buffer", DefaultHandshakeTriggerBuffer),

		messageMetrics: messageMetrics,
	}

	handshakeManager := NewHandshakeManager(l, tunCidr, preferredRanges, hostMap, lightHouse, udpConns[0], handshakeConfig, networkID)
	if handshakeManager.pendingHostMap.Hosts == nil {
		handshakeManager.l.WithField("pendingHostMap.Hosts[networkID] is ni...", networkID).Error("pendingHostMap1111")
	}
	if handshakeManager.pendingHostMap.Hosts[networkID] == nil {
		handshakeManager.l.WithField("pendingHostMap.Hosts[networkID] is ni...", networkID).Error("pendingHostMap1111222")
	}
	lightHouse.handshakeTrigger = handshakeManager.trigger

	//TODO: These will be reused for psk
	//handshakeMACKey := config.GetString("handshake_mac.key", "")
	//handshakeAcceptedMACKeys := config.GetStringSlice("handshake_mac.accepted_keys", []string{})

	serveDns := false
	if c.GetBool("lighthouse.serve_dns", false) {
		if c.GetBool("lighthouse.am_lighthouse", false) {
			serveDns = true
		} else {
			l.Warn("DNS server refusing to run because this host is not a lighthouse.")
		}
	}

	checkInterval := c.GetInt("timers.connection_alive_interval", 5)
	pendingDeletionInterval := c.GetInt("timers.pending_deletion_interval", 10)
	ifConfig := &InterfaceConfig{
		HostMap:                 hostMap,
		Inside:                  tun,
		Outside:                 udpConns[0],
		certState:               cs,
		Cipher:                  c.GetString("cipher", "aes"),
		Firewall:                fw,
		ServeDns:                serveDns,
		HandshakeManager:        handshakeManager,
		lightHouse:              lightHouse,
		checkInterval:           checkInterval,
		pendingDeletionInterval: pendingDeletionInterval,
		DropLocalBroadcast:      c.GetBool("tun.drop_local_broadcast", false),
		DropMulticast:           c.GetBool("tun.drop_multicast", false),
		routines:                routines,
		MessageMetrics:          messageMetrics,
		version:                 buildVersion,
		caFile:                  caFile,
		relayIndex:              relayIndex,
		sqlsecret:               c.GetString("lighthouse.sqlsecret", "SQL#secret123"),
		keysecret:               c.GetString("lighthouse.keysecret", "KEY#secret123"),
		disconnectInvalid:       c.GetBool("pki.disconnect_invalid", false),

		ConntrackCacheTimeout: conntrackCacheTimeout,
		l:                     l,
		relayServer:           relayServer,
		networkID:             networkID,
		Name:                  name,
	}

	switch ifConfig.Cipher {
	case "aes":
		noiseEndianness = binary.BigEndian
	case "chachapoly":
		noiseEndianness = binary.LittleEndian
	default:
		return nil, nil, fmt.Errorf("unknown cipher: %v", ifConfig.Cipher)
	}

	var ifce *Interface
	if !configTest {
		ifce, err = NewInterface(ctx, ifConfig, tunCidr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize interface: %s", err)
		}

		// TODO: Better way to attach these, probably want a new interface in InterfaceConfig
		// I don't want to make this initial commit too far-reaching though
		ifce.writers = udpConns

		ifce.RegisterConfigChangeCallbacks(c)

		go handshakeManager.Run(ctx, ifce)
		go lightHouse.LhUpdateWorker(ctx, ifce)
	}

	// TODO - stats third-party modules start uncancellable goroutines. Update those libs to accept
	// a context so that they can exit when the context is Done.
	statsStart, err := startStats(l, c, buildVersion, configTest)

	if err != nil {
		return nil, nil, util.NewContextualError("Failed to start stats emitter", nil, err)
	}

	if configTest {
		return nil, nil, nil
	}

	//TODO: check if we _should_ be emitting stats
	go ifce.emitStats(ctx, c.GetDuration("stats.interval", time.Second*10))
	mobilevpn := c.GetBool("tun.mobilevpn", false)
	if !mobilevpn {
		go ifce.checkDirectRoutesForRelayed(ctx)
	}

	attachCommands(l, ssh, hostMap, handshakeManager.pendingHostMap, lightHouse, ifce)

	// Start DNS server last to allow using the nebula IP as lighthouse.dns.host
	var dnsStart func()
	if amLighthouse && serveDns {
		l.Debugln("Starting dns server")
		dnsStart = dnsMain(l, hostMap, c)
	}

	return &Control{ifce, l, cancel, sshStart, statsStart, dnsStart, nil}, rs, nil
}