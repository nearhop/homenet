//go:build !router
// +build !router

package router

import (
	"github.com/sirupsen/logrus"
)

type RouterServer struct {
	a uint8
}

type RouterEvent struct {
	Active bool
}

func NewRouterServer(l1 *logrus.Logger) (*RouterServer, error) {
	rs := &RouterServer{}
	return rs, nil
}

func (rs *RouterServer) StartRouterServer() error {
	return nil
}

func (rs *RouterServer) GetNextEvent() *RouterEvent {
	return nil
}

func (rs *RouterServer) ShallUploadLogs() bool {
	return false
}

func (rs *RouterServer) MarkRouterEvent(event *RouterEvent, value bool) {
}

func (rs *RouterServer) GetRouterEventMessage(event *RouterEvent) ([]byte, error) {
	return nil, nil
}
