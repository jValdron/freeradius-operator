package v1

// LDAPTLSSpec configures TLS for LDAP connections.
type LDAPTLSSpec struct {
	// PEM CA bundle for validating the LDAP server certificate.
	// +optional
	CASecretRef *SecretRef `json:"caSecretRef,omitempty"`
}

// LDAPAuthMode selects how FreeRADIUS uses LDAP during inner-tunnel auth.
// +kubebuilder:validation:Enum=bindAsUser;mschap
type LDAPAuthMode string

const (
	LDAPAuthModeBindAsUser LDAPAuthMode = "bindAsUser"
	LDAPAuthModeMSCHAP     LDAPAuthMode = "mschap"
)

// LDAPSpec configures optional cluster-scoped LDAP for inner-tunnel authentication.
type LDAPSpec struct {
	// Hostnames or ldap(s):// URIs. Rendered as FreeRADIUS server = 'a b c'.
	// +kubebuilder:validation:MinItems=1
	Servers []string `json:"servers"`

	// Port applies to plain-hostname entries only (not ldap(s):// URIs).
	// +optional
	Port *int32 `json:"port,omitempty"`

	// +kubebuilder:validation:MinLength=1
	BaseDN string `json:"baseDN"`

	// +optional
	UserBaseDN string `json:"userBaseDN,omitempty"`

	// Bind DN for LDAP searches (may also be supplied via fromSecretRef key identity).
	// +optional
	Identity string `json:"identity,omitempty"`

	FromSecretRef *SecretRef `json:"fromSecretRef"`

	// +optional
	UserFilter string `json:"userFilter,omitempty"`

	// +kubebuilder:default=bindAsUser
	// +optional
	AuthMode LDAPAuthMode `json:"authMode,omitempty"`

	// LDAP attribute mapped to NT-Password (mschap authMode only).
	// +kubebuilder:default=ntPassword
	// +optional
	NTPasswordAttribute string `json:"ntPasswordAttribute,omitempty"`

	// When set, user filter is wrapped with (memberOf=<DN>).
	// +optional
	AccessGroupDN string `json:"accessGroupDN,omitempty"`

	// +optional
	StartTLS bool `json:"startTLS,omitempty"`

	// +optional
	TLS *LDAPTLSSpec `json:"tls,omitempty"`
}
