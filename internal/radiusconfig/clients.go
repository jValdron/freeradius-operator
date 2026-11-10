package radiusconfig

import (
	"fmt"
	"strings"
)

func buildClients(clients []ClientEntry) string {
	var b strings.Builder
	for _, c := range clients {
		fmt.Fprintf(&b, "client %s {\n", c.Name)
		fmt.Fprintf(&b, "\tipaddr = %s\n", c.IPAddress)
		fmt.Fprintf(&b, "\tsecret = \"%s\"\n", quote(c.Secret))
		fmt.Fprintf(&b, "}\n\n")
	}
	return b.String()
}
