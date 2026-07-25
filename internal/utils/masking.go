package utils

import (
	"net/netip"
	"strings"
)

// MaskString preserves a bounded prefix and suffix while replacing the middle.
func MaskString(value string, visiblePrefix, visibleSuffix int) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	if visiblePrefix < 0 {
		visiblePrefix = 0
	}
	if visibleSuffix < 0 {
		visibleSuffix = 0
	}
	if visiblePrefix+visibleSuffix >= len(runes) {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:visiblePrefix]) + strings.Repeat("*", len(runes)-visiblePrefix-visibleSuffix) + string(runes[len(runes)-visibleSuffix:])
}

func MaskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok {
		return MaskString(email, 1, 1)
	}
	return MaskString(local, 1, 1) + "@" + domain
}

func MaskPhone(phone string) string {
	return MaskString(phone, 3, 4)
}

func MaskIP(ip string) string {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return MaskString(ip, 2, 2)
	}
	if addr.Is4() {
		parts := strings.Split(addr.String(), ".")
		parts[3] = "*"
		return strings.Join(parts, ".")
	}
	bits := 64
	if addr.BitLen() < bits {
		bits = addr.BitLen()
	}
	return netip.PrefixFrom(addr, bits).Masked().String()
}
