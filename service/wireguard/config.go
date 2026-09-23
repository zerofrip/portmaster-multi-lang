package wireguard

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"unicode"
)

type Config struct {
	PrivateKey string
	Addresses  []netip.Addr
	DNS        []netip.Addr
	ListenPort uint16
	MTU        int
	Peers      []Peer
}

type Peer struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          []netip.Prefix
	PersistentKeepalive uint16
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{MTU: 1420}
	section := ""
	var peer *Peer
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			switch section {
			case "interface":
				peer = nil
			case "peer":
				cfg.Peers = append(cfg.Peers, Peer{})
				peer = &cfg.Peers[len(cfg.Peers)-1]
			default:
				return nil, fmt.Errorf("line %d: unsupported section", lineNumber)
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", lineNumber)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		var err error
		switch section {
		case "interface":
			err = parseInterfaceField(cfg, key, value)
		case "peer":
			err = parsePeerField(peer, key, value)
		default:
			return nil, fmt.Errorf("line %d: field outside a section", lineNumber)
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read WireGuard config: %w", err)
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("interface private key is required")
	}
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("at least one interface address is required")
	}
	if len(cfg.Peers) == 0 {
		return nil, fmt.Errorf("at least one peer is required")
	}
	seenAllowedIPs := make(map[netip.Prefix]int)
	for i := range cfg.Peers {
		if cfg.Peers[i].PublicKey == "" {
			return nil, fmt.Errorf("peer %d public key is required", i+1)
		}
		if cfg.Peers[i].Endpoint == "" {
			return nil, fmt.Errorf("peer %d endpoint is required", i+1)
		}
		if len(cfg.Peers[i].AllowedIPs) == 0 {
			return nil, fmt.Errorf("peer %d requires at least one allowed IP", i+1)
		}
		for _, prefix := range cfg.Peers[i].AllowedIPs {
			if previous, exists := seenAllowedIPs[prefix]; exists {
				return nil, fmt.Errorf("allowed IP %s is duplicated by peers %d and %d", prefix, previous+1, i+1)
			}
			seenAllowedIPs[prefix] = i
		}
	}
	return cfg, nil
}

func parseInterfaceField(cfg *Config, key, value string) error {
	switch key {
	case "privatekey":
		decoded, err := decodeKey(value)
		if err != nil {
			return fmt.Errorf("invalid interface private key")
		}
		cfg.PrivateKey = decoded
	case "address":
		for _, item := range splitList(value) {
			prefix, err := netip.ParsePrefix(item)
			if err == nil {
				cfg.Addresses = append(cfg.Addresses, prefix.Addr())
				continue
			}
			addr, err := netip.ParseAddr(item)
			if err != nil {
				return fmt.Errorf("invalid interface address")
			}
			cfg.Addresses = append(cfg.Addresses, addr)
		}
	case "dns":
		for _, item := range splitList(value) {
			addr, err := netip.ParseAddr(item)
			if err != nil {
				return fmt.Errorf("DNS entries must be IP addresses")
			}
			cfg.DNS = append(cfg.DNS, addr)
		}
	case "listenport":
		port, err := parseUint16(value)
		if err != nil {
			return fmt.Errorf("invalid listen port")
		}
		cfg.ListenPort = port
	case "mtu":
		mtu, err := strconv.Atoi(value)
		if err != nil || mtu < 1280 || mtu > 65535 {
			return fmt.Errorf("invalid MTU")
		}
		cfg.MTU = mtu
	case "table":
		// Route-table configuration is intentionally irrelevant to the userspace
		// netstack path.
	case "preup", "postup", "predown", "postdown", "saveconfig":
		return fmt.Errorf("script and stateful fields are not supported")
	default:
		return fmt.Errorf("unsupported interface field %q", key)
	}
	return nil
}

func parsePeerField(peer *Peer, key, value string) error {
	if peer == nil {
		return fmt.Errorf("peer field outside peer section")
	}
	switch key {
	case "publickey":
		decoded, err := decodeKey(value)
		if err != nil {
			return fmt.Errorf("invalid peer public key")
		}
		peer.PublicKey = decoded
	case "presharedkey":
		decoded, err := decodeKey(value)
		if err != nil {
			return fmt.Errorf("invalid peer preshared key")
		}
		peer.PresharedKey = decoded
	case "endpoint":
		if _, _, err := net.SplitHostPort(value); err != nil {
			return fmt.Errorf("invalid peer endpoint")
		}
		peer.Endpoint = value
	case "allowedips":
		for _, item := range splitList(value) {
			prefix, err := netip.ParsePrefix(item)
			if err != nil {
				return fmt.Errorf("invalid allowed IP")
			}
			peer.AllowedIPs = append(peer.AllowedIPs, prefix.Masked())
		}
	case "persistentkeepalive":
		keepalive, err := parseUint16(value)
		if err != nil {
			return fmt.Errorf("invalid persistent keepalive")
		}
		peer.PersistentKeepalive = keepalive
	default:
		return fmt.Errorf("unsupported peer field %q", key)
	}
	return nil
}

func decodeKey(value string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("WireGuard keys must decode to 32 bytes")
	}
	return hex.EncodeToString(decoded), nil
}

func splitList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func stripInlineComment(line string) string {
	if index := strings.IndexAny(line, "#;"); index >= 0 {
		return line[:index]
	}
	return line
}

func parseUint16(value string) (uint16, error) {
	parsed, err := strconv.ParseUint(value, 10, 16)
	return uint16(parsed), err
}

func (cfg *Config) UAPI() string {
	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\n", cfg.PrivateKey)
	if cfg.ListenPort != 0 {
		fmt.Fprintf(&b, "listen_port=%d\n", cfg.ListenPort)
	}
	b.WriteString("replace_peers=true\n")
	for _, peer := range cfg.Peers {
		fmt.Fprintf(&b, "public_key=%s\n", peer.PublicKey)
		if peer.PresharedKey != "" {
			fmt.Fprintf(&b, "preshared_key=%s\n", peer.PresharedKey)
		}
		if peer.Endpoint != "" {
			fmt.Fprintf(&b, "endpoint=%s\n", peer.Endpoint)
		}
		b.WriteString("replace_allowed_ips=true\n")
		for _, prefix := range peer.AllowedIPs {
			fmt.Fprintf(&b, "allowed_ip=%s\n", prefix)
		}
		if peer.PersistentKeepalive != 0 {
			fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", peer.PersistentKeepalive)
		}
	}
	return b.String()
}
