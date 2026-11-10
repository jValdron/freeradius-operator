package certs

import (
	"fmt"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

// Mode indicates how TLS material reaches the FreeRADIUS pod.
type Mode int

const (
	ModeInline Mode = iota
	ModeLegacySecret
	ModeCertManager
)

// Layout describes resolved certificate sources for a cluster.
type Layout struct {
	Mode Mode

	// Legacy / inline material copied into the generated Secret.
	Material *Material

	// Cert-manager direct mounts (secret names in cluster namespace unless overridden).
	TLSSecret freeradiusv1.SecretRef
	DHSecret  freeradiusv1.SecretRef
}

// Material is PEM bytes for the generated certs Secret.
type Material struct {
	CAPEM       []byte
	DH          []byte
	ServerKey   []byte
	ServerPEM   []byte
	KeyPassword string
}

// LayoutFromCluster picks certificate mode from spec (does not fetch secrets).
func LayoutFromCluster(spec freeradiusv1.CertificateSpec) (Layout, error) {
	switch {
	case spec.TLSSecretRef != nil && spec.TLSSecretRef.Name != "":
		layout := Layout{Mode: ModeCertManager, TLSSecret: *spec.TLSSecretRef}
		if spec.DHSecretRef != nil {
			layout.DHSecret = *spec.DHSecretRef
		}
		if layout.DHSecret.Name == "" && spec.Certificates != nil && len(spec.Certificates.DH) > 0 {
			// allow inline dh with tls secret ref
		} else if layout.DHSecret.Name == "" && spec.FromSecretRef == nil && spec.Certificates == nil {
			return Layout{}, fmt.Errorf("dhSecretRef is required when using tlsSecretRef")
		}
		return layout, nil
	case spec.FromSecretRef != nil && spec.FromSecretRef.Name != "":
		return Layout{Mode: ModeLegacySecret}, nil
	case spec.Certificates != nil:
		m := &Material{
			CAPEM:       []byte(spec.Certificates.CA),
			DH:          []byte(spec.Certificates.DH),
			ServerKey:   []byte(spec.Certificates.PrivateKey),
			ServerPEM:   []byte(spec.Certificates.PublicKey),
			KeyPassword: spec.PrivateKeyPassword,
		}
		return Layout{Mode: ModeInline, Material: m}, nil
	default:
		return Layout{}, fmt.Errorf("certificate: specify tlsSecretRef, fromSecretRef, or inline certificates")
	}
}
