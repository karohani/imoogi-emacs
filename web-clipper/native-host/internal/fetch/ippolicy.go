// Package fetch downloads clip images with SSRF-safe networking and validates
// them by magic bytes and header-only decoding.
package fetch

import "net/netip"

func pfx(s string) netip.Prefix { return netip.MustParsePrefix(s) }

var blocked4 = []netip.Prefix{
	pfx("0.0.0.0/8"),       // "this network"
	pfx("10.0.0.0/8"),      // private
	pfx("100.64.0.0/10"),   // CGNAT
	pfx("127.0.0.0/8"),     // loopback
	pfx("169.254.0.0/16"),  // link-local (cloud metadata)
	pfx("172.16.0.0/12"),   // private
	pfx("192.0.0.0/24"),    // IETF protocol assignments
	pfx("192.0.2.0/24"),    // TEST-NET-1
	pfx("192.88.99.0/24"),  // deprecated 6to4 relay anycast
	pfx("192.168.0.0/16"),  // private
	pfx("198.18.0.0/15"),   // benchmarking
	pfx("198.51.100.0/24"), // TEST-NET-2
	pfx("203.0.113.0/24"),  // TEST-NET-3
	pfx("224.0.0.0/4"),     // multicast
	pfx("240.0.0.0/4"),     // reserved + limited broadcast
}

var (
	globalUnicast6 = pfx("2000::/3")
	nat64          = pfx("64:ff9b::/96")
	sixToFour      = pfx("2002::/16")
	compat4        = pfx("::/96")
	blocked6       = []netip.Prefix{
		pfx("2001::/23"),     // IETF protocol assignments: Teredo, benchmarking, ORCHID
		pfx("2001:db8::/32"), // documentation
		pfx("3fff::/20"),     // documentation (RFC 9637)
	}
)

// IsPublic reports whether a is a globally routable unicast address that the
// downloader may connect to. It is a pure function of the address.
//
// IPv4-mapped addresses are unmapped and re-checked; NAT64 (64:ff9b::/96) and
// 6to4 (2002::/16) addresses are allowed only when the embedded IPv4 address
// is itself public. IPv4-compatible (::/96) and Teredo addresses are always
// refused. Other IPv6 must lie in 2000::/3 outside the special ranges.
//
// @MX:ANCHOR: [AUTO] SSRF policy evaluated on every connect (including redirect hops)
// @MX:REASON: the dialer Control hook and the up-front IP-literal check both depend on this; DNS-rebinding defence rests on it
func IsPublic(a netip.Addr) bool {
	if !a.IsValid() || a.Zone() != "" {
		return false
	}
	a = a.Unmap()
	if a.Is4() {
		for _, p := range blocked4 {
			if p.Contains(a) {
				return false
			}
		}
		return true
	}
	switch {
	case nat64.Contains(a):
		b := a.As16()
		return IsPublic(netip.AddrFrom4([4]byte(b[12:16])))
	case compat4.Contains(a): // includes :: and ::1
		return false
	case sixToFour.Contains(a):
		b := a.As16()
		return IsPublic(netip.AddrFrom4([4]byte(b[2:6])))
	case !globalUnicast6.Contains(a):
		return false
	}
	for _, p := range blocked6 {
		if p.Contains(a) {
			return false
		}
	}
	return true
}
