package radiusconfig

import (
	"fmt"
	"strings"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

func buildLDAP(cfg LDAPConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ldap {\n")
	fmt.Fprintf(&b, "\tserver = %q\n", cfg.ServerLine)
	if cfg.Port != nil {
		fmt.Fprintf(&b, "\tport = %d\n", *cfg.Port)
	}
	fmt.Fprintf(&b, "\tidentity = %q\n", cfg.Identity)
	fmt.Fprintf(&b, "\tpassword = %q\n", cfg.Password)
	fmt.Fprintf(&b, "\tbase_dn = %q\n\n", cfg.BaseDN)

	fmt.Fprintf(&b, "\tuser {\n")
	fmt.Fprintf(&b, "\t\tbase_dn = %q\n", cfg.UserBaseDN)
	fmt.Fprintf(&b, "\t\tfilter = %q\n", cfg.UserFilter)
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tupdate {\n")
	if cfg.AuthMode == freeradiusv1.LDAPAuthModeMSCHAP {
		fmt.Fprintf(&b, "\t\tcontrol:NT-Password := %q\n", cfg.NTPasswordAttribute)
	}
	fmt.Fprintf(&b, "\t}\n")

	if cfg.StartTLS || cfg.LDAPS || cfg.MountLdapCA {
		fmt.Fprintf(&b, "\n\ttls {\n")
		if cfg.StartTLS {
			fmt.Fprintf(&b, "\t\tstart_tls = yes\n")
		}
		if cfg.MountLdapCA {
			fmt.Fprintf(&b, "\t\tca_file = %q\n", LDAPCAFile)
		}
		fmt.Fprintf(&b, "\t}\n")
	}
	fmt.Fprintf(&b, "}\n")
	return b.String()
}
