//go:build !(screen || router) || server
// +build !screen,!router server

package screen

import (
	"github.com/slackhq/nebula/config"
)

type MainWindow struct {
	a uint8
}

var CommandCallback GUICommandCallback

func NewMainWindow() (*MainWindow, error) {
	return nil, nil
}

func (m *MainWindow) StartMainWindow(onboarded bool, c *config.C, cert string, status_err string, callback GUICommandCallback) error {
	return nil
}

func (m *MainWindow) SetHomeDetails(hd *HomeDetails) {
}

func (m *MainWindow) Onboarded(ip string) {
}
