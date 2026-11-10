package certs

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

func secretNamespace(clusterNS string, ref freeradiusv1.SecretRef) string {
	if ref.Namespace != "" {
		return ref.Namespace
	}
	return clusterNS
}

// ResolveMaterial loads certificate bytes for legacy/inline/dh resolution.
func ResolveMaterial(ctx context.Context, c client.Client, clusterNS string, spec freeradiusv1.CertificateSpec) (*Material, error) {
	layout, err := LayoutFromCluster(spec)
	if err != nil {
		return nil, err
	}

	switch layout.Mode {
	case ModeInline:
		return layout.Material, nil
	case ModeLegacySecret:
		ref := spec.FromSecretRef
		if ref == nil {
			return nil, fmt.Errorf("fromSecretRef is required")
		}
		return readLegacySecret(ctx, c, secretNamespace(clusterNS, *ref), ref.Name)
	case ModeCertManager:
		var mat *Material
		if spec.DHSecretRef != nil && spec.DHSecretRef.Name != "" {
			dh, err := readDHSecret(ctx, c, secretNamespace(clusterNS, *spec.DHSecretRef), spec.DHSecretRef.Name)
			if err != nil {
				return nil, err
			}
			mat = &Material{DH: dh}
		} else if spec.Certificates != nil && spec.Certificates.DH != "" {
			mat = &Material{DH: []byte(spec.Certificates.DH)}
		}
		if spec.TLSSecretRef != nil {
			// Optional CA from tls secret
			ca, err := readCAFromTLSSecret(ctx, c, secretNamespace(clusterNS, *spec.TLSSecretRef), spec.TLSSecretRef.Name)
			if err == nil && len(ca) > 0 {
				if mat == nil {
					mat = &Material{}
				}
				mat.CAPEM = ca
			}
		}
		return mat, nil
	default:
		return nil, nil
	}
}

func readLegacySecret(ctx context.Context, c client.Client, ns, name string) (*Material, error) {
	var sec corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &sec); err != nil {
		return nil, fmt.Errorf("certificate secret %s/%s: %w", ns, name, err)
	}
	m := &Material{
		CAPEM:     sec.Data["ca"],
		DH:        firstKey(sec.Data, "dh", "dh.pem"),
		ServerKey: firstKey(sec.Data, "privateKey", "tls.key", "server.key"),
		ServerPEM: firstKey(sec.Data, "publicKey", "tls.crt", "server.pem"),
	}
	if pw := sec.Data["privateKeyPassword"]; len(pw) > 0 {
		m.KeyPassword = string(pw)
	}
	if len(m.CAPEM) == 0 || len(m.ServerKey) == 0 || len(m.ServerPEM) == 0 {
		return nil, fmt.Errorf("certificate secret %s/%s missing ca, privateKey, or publicKey", ns, name)
	}
	return m, nil
}

func readDHSecret(ctx context.Context, c client.Client, ns, name string) ([]byte, error) {
	var sec corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &sec); err != nil {
		return nil, fmt.Errorf("dh secret %s/%s: %w", ns, name, err)
	}
	b := firstKey(sec.Data, "dh", "dh.pem")
	if len(b) == 0 {
		return nil, fmt.Errorf("dh secret %s/%s has no dh data", ns, name)
	}
	return b, nil
}

func readCAFromTLSSecret(ctx context.Context, c client.Client, ns, name string) ([]byte, error) {
	var sec corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &sec); err != nil {
		return nil, err
	}
	if ca := sec.Data["ca.crt"]; len(ca) > 0 {
		return ca, nil
	}
	return nil, nil
}

func firstKey(data map[string][]byte, keys ...string) []byte {
	for _, k := range keys {
		if b := data[k]; len(b) > 0 {
			return b
		}
	}
	return nil
}
