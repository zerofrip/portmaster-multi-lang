package wireguard

import (
	"strings"
	"testing"
)

const testKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestParseConfig(t *testing.T) {
	raw := `[Interface]
PrivateKey = ` + testKey + `
Address = 10.7.0.2/32, fd00::2/128
DNS = 10.7.0.1
MTU = 1380

[Peer]
PublicKey = ` + testKey + `
PresharedKey = ` + testKey + `
Endpoint = vpn.example.test:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.MTU != 1380 || len(cfg.Addresses) != 2 || len(cfg.Peers) != 1 {
		t.Fatalf("unexpected parsed config: %+v", cfg)
	}
	uapi := cfg.UAPI()
	if strings.Contains(uapi, testKey) {
		t.Fatal("UAPI must use decoded hexadecimal keys")
	}
	for _, expected := range []string{
		"private_key=" + strings.Repeat("0", 64),
		"public_key=" + strings.Repeat("0", 64),
		"allowed_ip=0.0.0.0/0",
		"allowed_ip=::/0",
		"endpoint=vpn.example.test:51820",
	} {
		if !strings.Contains(uapi, expected) {
			t.Fatalf("UAPI missing %q", expected)
		}
	}
}

func TestParseConfigRejectsSecretsInErrors(t *testing.T) {
	secret := "this-is-not-a-valid-private-key"
	_, err := ParseConfig("[Interface]\nPrivateKey = " + secret + "\nAddress = 10.0.0.2/32")
	if err == nil {
		t.Fatal("expected invalid key error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("error leaked secret input")
	}
}

func TestParseConfigRejectsScripts(t *testing.T) {
	raw := "[Interface]\nPrivateKey = " + testKey + "\nAddress = 10.0.0.2/32\nPostUp = calc.exe"
	if _, err := ParseConfig(raw); err == nil {
		t.Fatal("expected script field to be rejected")
	}
}

func TestParseConfigAddressCompatibility(t *testing.T) {
	raw := `[Interface]
PrivateKey = ` + testKey + ` # inline comment
Address = 10.7.0.2/32 fd00::2/128
Address = 192.0.2.10 ; bare address

[Peer]
PublicKey = ` + testKey + `
Endpoint = vpn.example.test:51820
AllowedIPs = 0.0.0.0/0 ::/0
`
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if len(cfg.Addresses) != 3 {
		t.Fatalf("expected three interface addresses, got %d", len(cfg.Addresses))
	}
	if len(cfg.Peers[0].AllowedIPs) != 2 {
		t.Fatalf("expected two allowed IPs, got %d", len(cfg.Peers[0].AllowedIPs))
	}
}
