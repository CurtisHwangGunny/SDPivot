package utils

import (
	"fmt"
	"net/netip"
	"strings"
)

// NormalizeIPWhitelist validates, canonicalizes, and de-duplicates IP and CIDR entries.
func NormalizeIPWhitelist(entries []string) ([]string, error) {
	out := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, raw := range entries {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		canonical := ""
		if prefix, err := netip.ParsePrefix(raw); err == nil {
			canonical = prefix.Masked().String()
		} else if addr, err := netip.ParseAddr(raw); err == nil {
			canonical = addr.Unmap().String()
		} else {
			return nil, fmt.Errorf("%q is not a valid IP address or CIDR", raw)
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out, nil
}

// IPAllowed reports whether an IP belongs to an exact-address or CIDR whitelist.
func IPAllowed(ip string, entries []string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, entry := range entries {
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			if prefix.Contains(addr) {
				return true
			}
			continue
		}
		if allowed, err := netip.ParseAddr(entry); err == nil && allowed.Unmap() == addr {
			return true
		}
	}
	return false
}
