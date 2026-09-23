// Package tunnel owns the provider-neutral tunnel mode and provider registry.
package tunnel

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/service/network"
)

const (
	CfgModeKey = "network/tunnel/mode"

	ModeOff       Mode = "off"
	ModeSPN       Mode = "spn"
	ModeWireGuard Mode = "wireguard"
)

type Mode string

func (m Mode) Valid() bool {
	return m == ModeOff || m == ModeSPN || m == ModeWireGuard
}

// Provider is an exclusive backend for connections entering the shared
// Portmaster tunnel listener.
type Provider interface {
	Mode() Mode
	Start(*mgr.WorkerCtx) error
	Stop(*mgr.WorkerCtx) error
	Ready() bool
	IsExcepted(net.IP) bool
	Handle(*network.Connection, net.Conn)
}

var (
	providersLock sync.RWMutex
	providers     = make(map[Mode]Provider)
	selectedMode  atomic.Value
)

func init() {
	selectedMode.Store(ModeOff)
}

func RegisterConfig() error {
	return config.Register(&config.Option{
		Name:         "Tunnel Mode",
		Key:          CfgModeKey,
		Description:  "Select the exclusive tunnel provider used for application traffic.",
		OptType:      config.OptTypeString,
		DefaultValue: string(ModeOff),
		PossibleValues: []config.PossibleValue{
			{Name: "Off", Value: string(ModeOff)},
			{Name: "SPN", Value: string(ModeSPN)},
			{Name: "WireGuard", Value: string(ModeWireGuard)},
		},
		Annotations: config.Annotations{
			config.CategoryAnnotation:    "General",
			config.DisplayHintAnnotation: config.DisplayHintOneOf,
		},
		ValidationFunc: func(value interface{}) error {
			mode, ok := value.(string)
			if !ok || !Mode(mode).Valid() {
				return fmt.Errorf("invalid tunnel mode %q", mode)
			}
			return nil
		},
	})
}

func ConfiguredMode() Mode {
	mode := Mode(config.GetAsString(CfgModeKey, string(ModeOff))())
	if !mode.Valid() {
		return ModeOff
	}
	return mode
}

// SelectedMode is the last mode accepted by the serialized transition manager.
func SelectedMode() Mode {
	return selectedMode.Load().(Mode)
}

func registerProvider(provider Provider) error {
	if provider == nil || provider.Mode() == ModeOff || !provider.Mode().Valid() {
		return errors.New("invalid tunnel provider")
	}
	providersLock.Lock()
	defer providersLock.Unlock()
	if _, exists := providers[provider.Mode()]; exists {
		return fmt.Errorf("tunnel provider %s already registered", provider.Mode())
	}
	providers[provider.Mode()] = provider
	return nil
}

// RegisterProvider registers a backend before the service starts.
func RegisterProvider(provider Provider) error {
	return registerProvider(provider)
}

func providerFor(mode Mode) Provider {
	providersLock.RLock()
	defer providersLock.RUnlock()
	return providers[mode]
}

func Ready() bool {
	mode := ConfiguredMode()
	if mode == ModeOff {
		return false
	}
	provider := providerFor(mode)
	return provider != nil && provider.Ready()
}

func IsExcepted(ip net.IP) bool {
	provider := providerFor(ConfiguredMode())
	return provider != nil && provider.IsExcepted(ip)
}

func Handle(connInfo *network.Connection, conn net.Conn) {
	provider := providerFor(ConfiguredMode())
	if provider == nil || !provider.Ready() {
		if conn != nil {
			_ = conn.Close()
		}
		connInfo.Lock()
		connInfo.Failed("selected tunnel provider is not ready", CfgModeKey)
		connInfo.SaveWhenFinished()
		connInfo.Unlock()
		return
	}
	provider.Handle(connInfo, conn)
}
