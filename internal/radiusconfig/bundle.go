package radiusconfig

import freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"

// UserEntry is a resolved RADIUS user (cleartext PAP).
type UserEntry struct {
	Name     string
	Username string
	Password string
}

// DeviceEntry maps MAC addresses to a VLAN for WPA enterprise.
type DeviceEntry struct {
	Name         string
	MACAddresses []string
	VLAN         int32
}

// ClientEntry is a RADIUS client (NAS).
type ClientEntry struct {
	Name      string
	IPAddress string
	Secret    string
}

// Input drives generation of FreeRADIUS configuration fragments.
type Input struct {
	ClusterName        string
	DefaultVlan        *int32
	EAP                *freeradiusv1.EAPSpec
	PrivateKeyPassword string
	// MountDHFile sets tls-config dh_file when custom DH params are mounted.
	MountDHFile bool
	LDAP        *LDAPConfig
	Users       []UserEntry
	Devices     []DeviceEntry
	Clients     []ClientEntry
}

// Bundle holds ConfigMap and mods key/value pairs.
type Bundle struct {
	Config map[string]string
	Mods   map[string]string
}

// BuildBundle renders authorize, clients.conf, radiusd.conf, and enabled mods.
func BuildBundle(in Input) Bundle {
	eap := MergeEAP(in.EAP)
	cfg := map[string]string{
		"authorize":    buildAuthorize(in.Users, in.Devices, in.DefaultVlan),
		"clients.conf": buildClients(in.Clients, in.LDAP != nil),
		"radiusd.conf": buildRadiusd(in.ClusterName),
	}
	mods := map[string]string{
		"eap":    buildEAP(eap, in.PrivateKeyPassword, in.MountDHFile),
		"mschap": buildMSCHAP(),
	}
	if in.LDAP != nil {
		ldapCfg := *in.LDAP
		mods["ldap"] = buildLDAP(ldapCfg)
		cfg["inner-tunnel"] = buildInnerTunnel(&ldapCfg)
	}
	return Bundle{Config: cfg, Mods: mods}
}
