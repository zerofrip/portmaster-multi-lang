package tunnel

import (
	"net"
	"testing"

	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/service/network"
)

func TestModeValidity(t *testing.T) {
	for _, mode := range []Mode{ModeOff, ModeSPN, ModeWireGuard} {
		if !mode.Valid() {
			t.Fatalf("expected %q to be valid", mode)
		}
	}
	if Mode("spn+wireguard").Valid() {
		t.Fatal("combined providers must never be a valid mode")
	}
}

type testProvider struct {
	mode    Mode
	stopped int
	stopErr error
}

func (p *testProvider) Mode() Mode                           { return p.mode }
func (p *testProvider) Start(*mgr.WorkerCtx) error           { return nil }
func (p *testProvider) Stop(*mgr.WorkerCtx) error            { p.stopped++; return p.stopErr }
func (p *testProvider) Ready() bool                          { return true }
func (p *testProvider) IsExcepted(net.IP) bool               { return false }
func (p *testProvider) Handle(*network.Connection, net.Conn) {}

func TestStopProvidersExceptStopsEveryOtherProvider(t *testing.T) {
	spn := &testProvider{mode: ModeSPN}
	wireguard := &testProvider{mode: ModeWireGuard}
	providersLock.Lock()
	previous := providers
	providers = map[Mode]Provider{ModeSPN: spn, ModeWireGuard: wireguard}
	providersLock.Unlock()
	t.Cleanup(func() {
		providersLock.Lock()
		providers = previous
		providersLock.Unlock()
	})

	if err := stopProvidersExcept(nil, ModeSPN); err != nil {
		t.Fatalf("stopProvidersExcept failed: %v", err)
	}
	if spn.stopped != 0 || wireguard.stopped != 1 {
		t.Fatalf("unexpected stop counts: SPN=%d WireGuard=%d", spn.stopped, wireguard.stopped)
	}
}
