package service

import (
	"net"

	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/service/network"
	"github.com/safing/portmaster/service/tunnel"
	"github.com/safing/portmaster/spn/captain"
	"github.com/safing/portmaster/spn/crew"
)

type spnTunnelProvider struct {
	instance *Instance
}

func (p spnTunnelProvider) Mode() tunnel.Mode         { return tunnel.ModeSPN }
func (p spnTunnelProvider) Ready() bool               { return captain.ClientReady() }
func (p spnTunnelProvider) IsExcepted(ip net.IP) bool { return captain.IsExcepted(ip) }
func (p spnTunnelProvider) Handle(info *network.Connection, conn net.Conn) {
	crew.HandleSluiceRequest(info, conn)
}
func (p spnTunnelProvider) Start(wc *mgr.WorkerCtx) error {
	return p.instance.SPNGroup().EnsureStartedWorker(wc)
}
func (p spnTunnelProvider) Stop(wc *mgr.WorkerCtx) error {
	return p.instance.SPNGroup().EnsureStoppedWorker(wc)
}
