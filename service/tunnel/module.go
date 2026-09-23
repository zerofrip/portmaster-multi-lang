package tunnel

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/base/log"
	"github.com/safing/portmaster/service/mgr"
)

type Manager struct {
	mgr      *mgr.Manager
	instance instance

	transitionLock sync.Mutex
	lastMode       Mode
	lastLegacySPN  bool
}

func (m *Manager) Manager() *mgr.Manager { return m.mgr }

func (m *Manager) Start() error {
	if err := m.migrateLegacySetting(); err != nil {
		return err
	}
	m.lastMode = ConfiguredMode()
	m.lastLegacySPN = config.GetAsBool("spn/enable", false)()
	m.instance.Config().EventConfigChange.AddCallback("exclusive tunnel mode", func(_ *mgr.WorkerCtx, _ struct{}) (bool, error) {
		if m.instance.IsShuttingDown() {
			return true, nil
		}
		m.mgr.Go("reconcile tunnel mode", m.reconcile)
		return false, nil
	})
	m.mgr.Go("start selected tunnel provider", m.reconcile)
	return nil
}

func (m *Manager) Stop() error {
	return m.mgr.Do("stop tunnel providers", func(wc *mgr.WorkerCtx) error {
		m.transitionLock.Lock()
		defer m.transitionLock.Unlock()
		selectedMode.Store(ModeOff)
		return stopProvidersExcept(wc, ModeOff)
	})
}

func (m *Manager) migrateLegacySetting() error {
	option, err := config.GetOption(CfgModeKey)
	if err != nil {
		return err
	}
	if !option.IsSetByUser() && config.GetAsBool("spn/enable", false)() {
		return config.SetConfigOption(CfgModeKey, string(ModeSPN))
	}
	return nil
}

func (m *Manager) reconcile(wc *mgr.WorkerCtx) error {
	m.transitionLock.Lock()
	defer m.transitionLock.Unlock()

	mode := ConfiguredMode()
	legacySPN := config.GetAsBool("spn/enable", false)()
	modeChanged := mode != m.lastMode
	legacyChanged := legacySPN != m.lastLegacySPN

	// Preserve old clients that only write spn/enable. If both settings changed,
	// the canonical mode wins.
	if legacyChanged && !modeChanged {
		if legacySPN {
			mode = ModeSPN
		} else if mode == ModeSPN {
			mode = ModeOff
		}
		m.lastMode = mode
		m.lastLegacySPN = legacySPN
		if err := config.SetConfigOption(CfgModeKey, string(mode)); err != nil {
			return err
		}
	} else if legacySPN != (mode == ModeSPN) {
		m.lastMode = mode
		m.lastLegacySPN = mode == ModeSPN
		if err := config.SetConfigOption("spn/enable", mode == ModeSPN); err != nil {
			return err
		}
	} else {
		m.lastMode = mode
		m.lastLegacySPN = legacySPN
	}

	// Publish the selected mode before transitions. Firewall traffic therefore
	// fails closed against the new provider until it reports Ready.
	selectedMode.Store(mode)
	if err := stopProvidersExcept(wc, mode); err != nil {
		return err
	}
	if mode == ModeOff {
		return nil
	}
	provider := providerFor(mode)
	if provider == nil {
		return errors.New("selected tunnel provider is unavailable")
	}
	if provider.Ready() {
		return nil
	}
	log.Infof("tunnel: starting exclusive provider %s", mode)
	return provider.Start(wc)
}

func stopProvidersExcept(wc *mgr.WorkerCtx, keep Mode) error {
	providersLock.RLock()
	list := make([]Provider, 0, len(providers))
	for mode, provider := range providers {
		if mode != keep {
			list = append(list, provider)
		}
	}
	providersLock.RUnlock()
	var stopErrors []error
	for _, provider := range list {
		if err := provider.Stop(wc); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("stop %s provider: %w", provider.Mode(), err))
		}
	}
	return errors.Join(stopErrors...)
}

var managerLoaded atomic.Bool

func New(instance instance) (*Manager, error) {
	if !managerLoaded.CompareAndSwap(false, true) {
		return nil, errors.New("only one tunnel manager instance allowed")
	}
	if err := RegisterConfig(); err != nil {
		return nil, err
	}
	return &Manager{mgr: mgr.New("Tunnel Manager"), instance: instance}, nil
}

type instance interface {
	Config() *config.Config
	IsShuttingDown() bool
}
