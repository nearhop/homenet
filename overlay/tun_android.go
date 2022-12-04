//go:build !e2e_testing
// +build !e2e_testing

package overlay

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cidr"
	"github.com/slackhq/nebula/iputil"
)

type tun struct {
	io.ReadWriteCloser
	fd         int
	cidr       *net.IPNet
	routeTree  *cidr.Tree4
	l          *logrus.Logger
	DNSServers []net.IP
}

func newTunFromFd(l *logrus.Logger, deviceFd int, cidr *net.IPNet, _ int, routes []Route, _ int, dns []net.IP) (*tun, error) {
	routeTree, err := makeRouteTree(l, routes, false)
	if err != nil {
		return nil, err
	}

	file := os.NewFile(uintptr(deviceFd), "/dev/net/tun")

	return &tun{
		ReadWriteCloser: file,
		fd:              int(file.Fd()),
		cidr:            cidr,
		routeTree:       routeTree,
		l:               l,
		DNSServers:      dns,
	}, nil
}

func newTun(_ *logrus.Logger, _ string, _ *net.IPNet, _ int, _ []Route, _ int, _ bool, dns []net.IP) (*tun, error) {
	return nil, fmt.Errorf("newTun not supported in Android")
}

func (t *tun) RouteFor(ip iputil.VpnIp) iputil.VpnIp {
	r := t.routeTree.MostSpecificContains(ip)
	if r != nil {
		return r.(iputil.VpnIp)
	}

	return 0
}

func (t *tun) AddRoutes(Routes []Route) error {
	return nil
}

func (t tun) Activate() error {
	return nil
}

func (t *tun) Cidr() *net.IPNet {
	return t.cidr
}

func (t *tun) Name() string {
	return "android"
}

func (t *tun) NewMultiQueueReader() (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("TODO: multiqueue not implemented for android")
}

func (t *tun) GetPlatformName() string {
	return "android"
}
