package certs

import freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"

// MountDHParams reports whether FreeRADIUS should use dh_file (custom parameters).
// When false, OpenSSL selects built-in DH parameters (recommended for cert-manager TLS).
func MountDHParams(spec freeradiusv1.CertificateSpec, layout Layout, mat *Material) bool {
	if spec.DHSecretRef != nil && spec.DHSecretRef.Name != "" {
		return true
	}
	if spec.Certificates != nil && spec.Certificates.DH != "" {
		return true
	}
	return mat != nil && len(mat.DH) > 0
}
