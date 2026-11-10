package radiusconfig

import "strings"

// quote escapes a string for use inside double quotes in FreeRADIUS unlang assignments.
func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func normalizeMAC(mac string) string {
	return strings.NewReplacer(":", "", "-", "").Replace(mac)
}
