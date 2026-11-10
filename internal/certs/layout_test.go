package certs_test

import (
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
)

func TestLayoutFromClusterCertManagerWithoutDHRef(t *testing.T) {
	layout, err := certs.LayoutFromCluster(freeradiusv1.CertificateSpec{
		TLSSecretRef: &freeradiusv1.SecretRef{Name: "tls"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if layout.Mode != certs.ModeCertManager {
		t.Fatalf("mode %v", layout.Mode)
	}
}

func TestMountDHParamsCertManagerDefault(t *testing.T) {
	spec := freeradiusv1.CertificateSpec{TLSSecretRef: &freeradiusv1.SecretRef{Name: "tls"}}
	layout, err := certs.LayoutFromCluster(spec)
	if err != nil {
		t.Fatal(err)
	}
	if certs.MountDHParams(spec, layout, nil) {
		t.Fatal("expected no custom DH for cert-manager TLS only")
	}
}

func TestMountDHParamsWithDHSecretRef(t *testing.T) {
	spec := freeradiusv1.CertificateSpec{
		TLSSecretRef: &freeradiusv1.SecretRef{Name: "tls"},
		DHSecretRef:  &freeradiusv1.SecretRef{Name: "dh"},
	}
	layout, err := certs.LayoutFromCluster(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !certs.MountDHParams(spec, layout, nil) {
		t.Fatal("expected DH mount when dhSecretRef set")
	}
}
