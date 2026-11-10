package radiusconfig

import (
	"fmt"
	"strings"
)

func buildAuthorize(users []UserEntry, devices []DeviceEntry, defaultVlan *int32) string {
	var b strings.Builder
	for _, u := range users {
		user := u.Username
		if user == "" {
			user = u.Name
		}
		fmt.Fprintf(&b, "# %s\n", u.Name)
		fmt.Fprintf(&b, "%s\tCleartext-Password := \"%s\"\n\n", user, quote(u.Password))
	}
	for _, d := range devices {
		fmt.Fprintf(&b, "# %s\n", d.Name)
		for _, mac := range d.MACAddresses {
			norm := normalizeMAC(mac)
			fmt.Fprintf(&b, "%s\tCleartext-Password := \"%s\"\n", norm, quote(norm))
			fmt.Fprintf(&b, "\tTunnel-Type = 13,\n")
			fmt.Fprintf(&b, "\tTunnel-Medium-Type = 6,\n")
			fmt.Fprintf(&b, "\tTunnel-Private-Group-Id = %d\n\n", d.VLAN)
		}
	}
	if defaultVlan != nil {
		fmt.Fprintf(&b, "DEFAULT\tCleartext-Password := \"%%{User-Name}\"\n")
		fmt.Fprintf(&b, "\tTunnel-Type = 13,\n")
		fmt.Fprintf(&b, "\tTunnel-Medium-Type = 6,\n")
		fmt.Fprintf(&b, "\tTunnel-Private-Group-Id = %d\n", *defaultVlan)
	}
	return b.String()
}
