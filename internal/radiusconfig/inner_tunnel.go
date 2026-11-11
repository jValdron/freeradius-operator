package radiusconfig

import (
	_ "embed"
	"strings"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

// Stock inner-tunnel from freeradius/freeradius-server:3.2.10-alpine.
//
//go:embed embed/inner-tunnel.default
var innerTunnelDefault []byte

const (
	innerTunnelLDAPDisabled = "\t#\n\t#  The ldap module reads passwords from the LDAP database.\n\t-ldap"

	innerTunnelLDAPBindAsUser = "\t#\n\t#  The ldap module reads passwords from the LDAP database.\n\tldap\n\n\tif ((ok || updated) && User-Password) {\n\t\tupdate {\n\t\t\tcontrol:Auth-Type := ldap\n\t\t}\n\t}"

	innerTunnelLDAPMschap = "\t#\n\t#  The ldap module reads passwords from the LDAP database.\n\tldap"

	innerTunnelAuthTypeLDAPCommented = "#\tAuth-Type LDAP {\n#\t\tldap\n#\t}"

	innerTunnelAuthTypeLDAPEnabled = "\tAuth-Type LDAP {\n\t\tldap\n\t}"
)

func buildInnerTunnel(cfg *LDAPConfig) string {
	conf := string(innerTunnelDefault)
	if cfg == nil {
		return conf
	}

	switch cfg.AuthMode {
	case freeradiusv1.LDAPAuthModeMSCHAP:
		conf = strings.Replace(conf, innerTunnelLDAPDisabled, innerTunnelLDAPMschap, 1)
	default:
		conf = strings.Replace(conf, innerTunnelLDAPDisabled, innerTunnelLDAPBindAsUser, 1)
		conf = strings.Replace(conf, innerTunnelAuthTypeLDAPCommented, innerTunnelAuthTypeLDAPEnabled, 1)
	}
	return conf
}
