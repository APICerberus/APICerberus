package config

import "strings"

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
