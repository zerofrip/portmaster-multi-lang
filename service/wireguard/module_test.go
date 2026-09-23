package wireguard

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestResolvePeerEndpointsPrefersIPv4(t *testing.T) {
	cfg := &Config{Peers: []Peer{{Endpoint: "vpn.example.test:51820"}}}
	endpointIPs, err := resolvePeerEndpoints(context.Background(), cfg, func(_ context.Context, network, host string) ([]net.IP, error) {
		if network != "ip" || host != "vpn.example.test" {
			t.Fatalf("unexpected lookup: network=%q host=%q", network, host)
		}
		return []net.IP{net.ParseIP("2001:db8::1"), net.ParseIP("192.0.2.1")}, nil
	})
	if err != nil {
		t.Fatalf("resolvePeerEndpoints failed: %v", err)
	}
	if got := cfg.Peers[0].Endpoint; got != "192.0.2.1:51820" {
		t.Fatalf("unexpected resolved endpoint %q", got)
	}
	if len(endpointIPs) != 1 || !endpointIPs[0].Equal(net.ParseIP("192.0.2.1")) {
		t.Fatalf("unexpected endpoint exception list: %v", endpointIPs)
	}
}

func TestResolvePeerEndpointsRetainsNumericIPv6(t *testing.T) {
	cfg := &Config{Peers: []Peer{{Endpoint: "[2001:db8::1]:51820"}}}
	endpointIPs, err := resolvePeerEndpoints(context.Background(), cfg, func(context.Context, string, string) ([]net.IP, error) {
		t.Fatal("numeric endpoint must not use DNS lookup")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("resolvePeerEndpoints failed: %v", err)
	}
	if got := cfg.Peers[0].Endpoint; got != "[2001:db8::1]:51820" {
		t.Fatalf("unexpected endpoint %q", got)
	}
	if len(endpointIPs) != 1 || !endpointIPs[0].Equal(net.ParseIP("2001:db8::1")) {
		t.Fatalf("unexpected endpoint exception list: %v", endpointIPs)
	}
}

func TestResolvePeerEndpointsSanitizesLookupFailure(t *testing.T) {
	const secretHost = "private-endpoint.example.test"
	cfg := &Config{Peers: []Peer{{Endpoint: secretHost + ":51820"}}}
	_, err := resolvePeerEndpoints(context.Background(), cfg, func(context.Context, string, string) ([]net.IP, error) {
		return nil, errors.New("resolver included " + secretHost)
	})
	if err == nil {
		t.Fatal("expected endpoint lookup failure")
	}
	if got := err.Error(); got != "resolve endpoint for peer 1" {
		t.Fatalf("lookup error was not sanitized: %q", got)
	}
}

func TestResolvePeerEndpointsRejectsEmptyLookupResult(t *testing.T) {
	cfg := &Config{Peers: []Peer{{Endpoint: "vpn.example.test:51820"}}}
	_, err := resolvePeerEndpoints(context.Background(), cfg, func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{nil}, nil
	})
	if err == nil {
		t.Fatal("expected empty endpoint lookup result to fail")
	}
}
