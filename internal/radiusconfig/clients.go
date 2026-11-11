package radiusconfig

import (
	"fmt"
	"strings"
)

func buildClients(clients []ClientEntry, includeLocalhost bool) string {
	var b strings.Builder
	if includeLocalhost {
		// Stock image default: inner-tunnel listens on 127.0.0.1:18120 for in-pod radclient/radtest.
		fmt.Fprintf(&b, "client localhost {\n\tipaddr = 127.0.0.1\n\tsecret = \"testing123\"\n}\n\n")
	}
	for _, c := range clients {
		fmt.Fprintf(&b, "client %s {\n", c.Name)
		fmt.Fprintf(&b, "\tipaddr = %s\n", c.IPAddress)
		fmt.Fprintf(&b, "\tsecret = \"%s\"\n", quote(c.Secret))
		fmt.Fprintf(&b, "}\n\n")
	}
	return b.String()
}
