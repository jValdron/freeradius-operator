package v1

// SecretRef points at a Kubernetes Secret.
type SecretRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// InlineCertificates holds PEM material for legacy / bootstrap setups.
type InlineCertificates struct {
	// +kubebuilder:validation:MinLength=1
	CA string `json:"ca"`

	// +optional
	DH string `json:"dh,omitempty"`

	// +kubebuilder:validation:MinLength=1
	PrivateKey string `json:"privateKey"`

	// +kubebuilder:validation:MinLength=1
	PublicKey string `json:"publicKey"`
}

// CertificateSpec configures server TLS for EAP.
type CertificateSpec struct {
	// +optional
	PrivateKeyPassword string `json:"privateKeyPassword,omitempty"`

	// +optional
	Certificates *InlineCertificates `json:"certificates,omitempty"`

	// Legacy secret keys: ca, dh, privateKey, publicKey, privateKeyPassword.
	// +optional
	FromSecretRef *SecretRef `json:"fromSecretRef,omitempty"`

	// cert-manager TLS secret (tls.crt, tls.key, optional ca.crt).
	// +optional
	TLSSecretRef *SecretRef `json:"tlsSecretRef,omitempty"`

	// DH parameters when using TLSSecretRef (cert-manager does not supply dh).
	// +optional
	DHSecretRef *SecretRef `json:"dhSecretRef,omitempty"`
}

// ServiceSpec configures the generated Service.
type ServiceSpec struct {
	// +kubebuilder:validation:MinLength=1
	Type string `json:"type"`

	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
}

// EAPSpec tunes generated mods-enabled/eap (secure defaults).
type EAPSpec struct {
	// +kubebuilder:validation:Enum=ttls;peap;tls
	// +kubebuilder:default=ttls
	// +optional
	DefaultEapType string `json:"defaultEapType,omitempty"`

	// +kubebuilder:default="1.2"
	// +optional
	TLSMinVersion string `json:"tlsMinVersion,omitempty"`

	// +kubebuilder:default="1.3"
	// +optional
	TLSMaxVersion string `json:"tlsMaxVersion,omitempty"`

	// +kubebuilder:default=DEFAULT
	// +optional
	CipherList string `json:"cipherList,omitempty"`

	// +kubebuilder:default=inner-tunnel
	// +optional
	InnerVirtualServer string `json:"innerVirtualServer,omitempty"`
}
