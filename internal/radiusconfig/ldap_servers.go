package radiusconfig

import (
	"fmt"
	"strings"
)

// FormatLDAPServers renders the FreeRADIUS ldap module server directive value and optional port.
func FormatLDAPServers(servers []string, port *int32) (serverLine string, portOut *int32, err error) {
	if len(servers) == 0 {
		return "", nil, fmt.Errorf("ldap servers list is empty")
	}
	entries := make([]string, 0, len(servers))
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", nil, fmt.Errorf("ldap servers contains an empty entry")
		}
		entries = append(entries, s)
	}
	if port != nil {
		for _, s := range entries {
			if strings.Contains(s, "://") {
				return "", nil, fmt.Errorf("ldap port must not be set when any server is an ldap(s):// URI")
			}
		}
		return strings.Join(entries, " "), port, nil
	}
	return strings.Join(entries, " "), nil, nil
}
