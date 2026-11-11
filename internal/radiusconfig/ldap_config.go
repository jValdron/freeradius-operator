package radiusconfig

import freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"

// LDAPConfig holds resolved LDAP settings for FreeRADIUS config generation.
type LDAPConfig struct {
	ServerLine          string
	Port                *int32
	BaseDN              string
	UserBaseDN          string
	Identity            string
	Password            string
	UserFilter          string
	AuthMode            freeradiusv1.LDAPAuthMode
	NTPasswordAttribute string
	StartTLS            bool
	LDAPS               bool
	MountLdapCA         bool
	CASecret            freeradiusv1.SecretRef
}

// LDAPCAFile is where the operator mounts an optional LDAP CA bundle in the pod.
const LDAPCAFile = "/etc/raddb/certs/ldap-ca.pem"
