package utils

import (
	"net/netip"
	"strings"
)

var sensitiveMaskFields = map[string]struct{}{
	"authorization": {}, "password": {}, "password_hash": {}, "token": {},
	"access_token": {}, "refresh_token": {}, "api_key": {}, "apikey": {},
	"secret": {}, "client_secret": {}, "private_key": {},
}

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

// MaskSensitiveData recursively copies JSON-like data and replaces values of
// sensitive keys. The input maps and slices are never mutated.
func MaskSensitiveData(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			if isSensitiveMaskField(key) {
				out[key] = maskSensitiveValue(item)
			} else {
				out[key] = MaskSensitiveData(item)
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = MaskSensitiveData(item)
		}
		return out
	default:
		return value
	}
}

func isSensitiveMaskField(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")
	_, ok := sensitiveMaskFields[key]
	return ok
}

func maskSensitiveValue(value any) any {
	if value == nil {
		return nil
	}
	if s, ok := value.(string); ok {
		if s == "" {
			return ""
		}
		return "******"
	}
	return "******"
}
