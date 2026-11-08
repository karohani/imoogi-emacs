package fetch

import (
	"net/netip"
	"testing"
)

func TestIsPublic(t *testing.T) {
	cases := map[string]bool{
		// IPv4 public
		"8.8.8.8":         true,
		"1.1.1.1":         true,
		"93.184.216.34":   true,
		"100.63.255.255":  true,
		"100.128.0.0":     true,
		"172.15.255.255":  true,
		"172.32.0.0":      true,
		"192.0.1.1":       true,
		"198.17.255.255":  true,
		"223.255.255.255": true,
		// IPv4 blocked
		"0.0.0.0":         false,
		"0.1.2.3":         false,
		"10.0.0.1":        false,
		"100.64.0.1":      false,
		"100.127.255.255": false,
		"127.0.0.1":       false,
		"127.255.255.254": false,
		"169.254.169.254": false,
		"172.16.0.1":      false,
		"172.31.255.255":  false,
		"192.0.0.8":       false,
		"192.0.2.1":       false,
		"192.88.99.1":     false,
		"192.168.1.1":     false,
		"198.18.0.1":      false,
		"198.19.255.255":  false,
		"198.51.100.7":    false,
		"203.0.113.9":     false,
		"224.0.0.1":       false,
		"239.255.255.250": false,
		"240.0.0.1":       false,
		"255.255.255.255": false,
		// IPv6 public
		"2606:4700:4700::1111":     true,
		"2a00:1450:4001:80b::200e": true,
		// IPv6 blocked
		"::":                     false,
		"::1":                    false,
		"fe80::1":                false,
		"fe80::1%en0":            false,
		"fc00::1":                false,
		"fd12:3456::1":           false,
		"fec0::1":                false,
		"ff02::1":                false,
		"ff05::2":                false,
		"2001:db8::1":            false,
		"3fff::1":                false,
		"2001::1":                false, // Teredo
		"2001:2::1":              false, // benchmarking
		"100::1":                 false, // discard-only
		"::ffff:127.0.0.1":       false, // IPv4-mapped loopback
		"::ffff:10.0.0.1":        false,
		"::ffff:169.254.169.254": false,
		"::ffff:8.8.8.8":         true,  // mapped public: unmapped and re-checked
		"::127.0.0.1":            false, // IPv4-compatible (deprecated): always blocked
		"::8.8.8.8":              false,
		"64:ff9b::7f00:1":        false, // NAT64 of 127.0.0.1
		"64:ff9b::a00:1":         false, // NAT64 of 10.0.0.1
		"64:ff9b::808:808":       true,  // NAT64 of 8.8.8.8
		"64:ff9b:1::808:808":     false, // local-use NAT64
		"2002:7f00:1::1":         false, // 6to4 embedding 127.0.0.1
		"2002:c0a8:101::1":       false, // 6to4 embedding 192.168.1.1
		"2002:808:808::1":        true,  // 6to4 embedding 8.8.8.8
		"5f00::1":                false, // outside 2000::/3
	}
	for s, want := range cases {
		a, err := netip.ParseAddr(s)
		if err != nil {
			t.Fatalf("bad test address %s: %v", s, err)
		}
		if got := IsPublic(a); got != want {
			t.Errorf("IsPublic(%s) = %v, want %v", s, got, want)
		}
	}
	if IsPublic(netip.Addr{}) {
		t.Error("zero Addr reported public")
	}
}
