package wireguard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	wgconn "golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/base/log"
	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/service/network"
	"github.com/safing/portmaster/service/network/packet"
	"github.com/safing/portmaster/service/tunnel"
)

const CfgConfigKey = "wireguard/config"

type Module struct {
	mgr *mgr.Manager

	lock         sync.RWMutex
	device       *device.Device
	network      *netstack.Net
	endpointIPs  []net.IP
	activeConfig string
	ready        atomic.Bool
}

func (m *Module) Manager() *mgr.Manager { return m.mgr }
func (m *Module) Mode() tunnel.Mode     { return tunnel.ModeWireGuard }
func (m *Module) Start() error          { return nil }
func (m *Module) Stop() error           { return m.stopProvider() }

func (m *Module) StartProvider(wc *mgr.WorkerCtx) error {
	return m.startProvider(wc.Ctx())
}

// Provider methods use the worker-aware names through a small adapter because
// mgr.Module already reserves Start and Stop without a WorkerCtx.
type provider struct{ module *Module }

func (p provider) Mode() tunnel.Mode                           { return tunnel.ModeWireGuard }
func (p provider) Start(wc *mgr.WorkerCtx) error               { return p.module.startProvider(wc.Ctx()) }
func (p provider) Stop(_ *mgr.WorkerCtx) error                 { return p.module.stopProvider() }
func (p provider) Ready() bool                                 { return p.module.isReady() }
func (p provider) IsExcepted(ip net.IP) bool                   { return p.module.isExcepted(ip) }
func (p provider) Handle(info *network.Connection, c net.Conn) { p.module.handle(info, c) }

func New() (*Module, error) {
	m := &Module{mgr: mgr.New("WireGuard")}
	if err := registerConfig(); err != nil {
		return nil, err
	}
	if err := registerStatus(); err != nil {
		return nil, err
	}
	if err := tunnel.RegisterProvider(provider{module: m}); err != nil {
		return nil, err
	}
	return m, nil
}

func registerConfig() error {
	return config.Register(&config.Option{
		Name:         "WireGuard Configuration",
		Key:          CfgConfigKey,
		Description:  "A standard WireGuard configuration used by the Portmaster userspace tunnel.",
		Help:         "Paste a WireGuard configuration containing an Interface and at least one Peer.",
		Sensitive:    true,
		OptType:      config.OptTypeString,
		DefaultValue: "",
		Annotations: config.Annotations{
			config.CategoryAnnotation: "WireGuard",
		},
		ValidationFunc: func(value interface{}) error {
			raw, ok := value.(string)
			if !ok {
				return errors.New("WireGuard configuration must be a string")
			}
			if raw == "" {
				return nil
			}
			_, err := ParseConfig(raw)
			return err
		},
	})
}

func (m *Module) startProvider(ctx context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	raw := config.GetAsString(CfgConfigKey, "")()
	if m.ready.Load() && raw == m.activeConfig {
		return nil
	}
	if m.ready.Load() {
		m.stopProviderLocked()
	}
	setStatus("connecting", "", nil)
	cfg, err := ParseConfig(raw)
	if err != nil {
		setStatus("failed", err.Error(), nil)
		return fmt.Errorf("invalid WireGuard configuration: %w", err)
	}
	endpointIPs, err := resolvePeerEndpoints(ctx, cfg, net.DefaultResolver.LookupIP)
	if err != nil {
		setStatus("failed", "failed to resolve WireGuard endpoint", nil)
		return errors.New("failed to resolve WireGuard endpoint")
	}

	tunDevice, tunnelNetwork, err := netstack.CreateNetTUN(cfg.Addresses, cfg.DNS, cfg.MTU)
	if err != nil {
		setStatus("failed", "failed to create userspace network stack", nil)
		return fmt.Errorf("create WireGuard netstack: %w", err)
	}
	// The configuration passed to IpcSet contains private key material. Keep the
	// upstream device logger silent so malformed or future UAPI errors cannot
	// accidentally copy configuration values into Portmaster logs.
	wgDevice := device.NewDevice(tunDevice, wgconn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))
	if err = wgDevice.IpcSet(cfg.UAPI()); err != nil {
		wgDevice.Close()
		setStatus("failed", "failed to apply WireGuard configuration", nil)
		return errors.New("failed to apply WireGuard configuration")
	}
	if err = wgDevice.Up(); err != nil {
		wgDevice.Close()
		setStatus("failed", "failed to start WireGuard device", nil)
		return fmt.Errorf("start WireGuard device: %w", err)
	}

	m.device = wgDevice
	m.network = tunnelNetwork
	m.endpointIPs = endpointIPs
	m.activeConfig = raw
	m.ready.Store(true)
	now := time.Now()
	setStatus("connected", "", &now)
	return nil
}

func (m *Module) stopProvider() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.stopProviderLocked()
	return nil
}

func (m *Module) stopProviderLocked() {
	m.ready.Store(false)
	if m.device != nil {
		m.device.Close()
		m.device = nil
	}
	m.network = nil
	m.endpointIPs = nil
	m.activeConfig = ""
	setStatus("disabled", "", nil)
}

func (m *Module) isReady() bool {
	if !m.ready.Load() {
		return false
	}
	m.lock.RLock()
	activeConfig := m.activeConfig
	m.lock.RUnlock()
	return config.GetAsString(CfgConfigKey, "")() == activeConfig
}

type endpointLookupFunc func(context.Context, string, string) ([]net.IP, error)

func resolvePeerEndpoints(ctx context.Context, cfg *Config, lookup endpointLookupFunc) ([]net.IP, error) {
	endpointIPs := make([]net.IP, 0, len(cfg.Peers))
	for i := range cfg.Peers {
		host, port, err := net.SplitHostPort(cfg.Peers[i].Endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint for peer %d", i+1)
		}
		selectedIP := net.ParseIP(host)
		if selectedIP == nil {
			addresses, err := lookup(ctx, "ip", host)
			if err != nil || len(addresses) == 0 {
				return nil, fmt.Errorf("resolve endpoint for peer %d", i+1)
			}
			for _, address := range addresses {
				if selectedIP == nil && address != nil {
					selectedIP = address
				}
				if ipv4 := address.To4(); ipv4 != nil {
					selectedIP = ipv4
					break
				}
			}
			if selectedIP == nil {
				return nil, fmt.Errorf("resolve endpoint for peer %d", i+1)
			}
		}
		if ipv4 := selectedIP.To4(); ipv4 != nil {
			selectedIP = ipv4
		}
		cfg.Peers[i].Endpoint = net.JoinHostPort(selectedIP.String(), port)
		endpointIPs = append(endpointIPs, selectedIP)
	}
	return endpointIPs, nil
}

func (m *Module) isExcepted(ip net.IP) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, endpointIP := range m.endpointIPs {
		if endpointIP.Equal(ip) {
			return true
		}
	}
	return false
}

func (m *Module) handle(connInfo *network.Connection, entry net.Conn) {
	if entry == nil {
		return
	}
	m.mgr.Go("WireGuard tunnel connection", func(wc *mgr.WorkerCtx) error {
		m.lock.RLock()
		tunnelNetwork := m.network
		ready := m.ready.Load()
		m.lock.RUnlock()
		if !ready || tunnelNetwork == nil {
			_ = entry.Close()
			markConnectionFailed(connInfo, "WireGuard is not ready")
			return nil
		}

		networkName := ""
		switch connInfo.IPProtocol {
		case packet.TCP:
			networkName = "tcp"
		case packet.UDP:
			networkName = "udp"
		default:
			_ = entry.Close()
			markConnectionFailed(connInfo, "protocol is not supported by WireGuard")
			return nil
		}
		if connInfo.Entity.IP.To4() != nil {
			networkName += "4"
		} else {
			networkName += "6"
		}
		destination := net.JoinHostPort(connInfo.Entity.IP.String(), strconv.Itoa(int(connInfo.Entity.Port)))
		exit, err := tunnelNetwork.DialContext(wc.Ctx(), networkName, destination)
		if err != nil {
			_ = entry.Close()
			markConnectionFailed(connInfo, "WireGuard failed to connect to destination")
			return nil
		}

		ctx := &connectionContext{entry: entry, exit: exit}
		connInfo.Lock()
		connInfo.TunnelContext = ctx
		connInfo.Save()
		connInfo.Unlock()

		done := make(chan struct{}, 2)
		copyConn := func(dst, src net.Conn) {
			_, _ = io.Copy(dst, src)
			done <- struct{}{}
		}
		go copyConn(exit, entry)
		go copyConn(entry, exit)
		select {
		case <-done:
		case <-wc.Done():
		}
		_ = ctx.StopTunnel()
		return nil
	})
}

func markConnectionFailed(connInfo *network.Connection, reason string) {
	connInfo.Lock()
	defer connInfo.Unlock()
	connInfo.Failed(reason, CfgConfigKey)
	connInfo.Save()
	log.Debugf("wireguard: %s for %s", reason, connInfo)
}

type connectionContext struct {
	lock  sync.Mutex
	entry net.Conn
	exit  net.Conn
}

func (c *connectionContext) GetExitNodeID() string { return "wireguard" }

func (c *connectionContext) StopTunnel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	var firstErr error
	if c.entry != nil {
		firstErr = c.entry.Close()
		c.entry = nil
	}
	if c.exit != nil {
		if err := c.exit.Close(); firstErr == nil {
			firstErr = err
		}
		c.exit = nil
	}
	return firstErr
}
