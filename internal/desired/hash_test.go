package desired

import (
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestConfigHashStableAcrossEquivalentBundleMaps(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		Name: "primary", Namespace: "default",
		Spec: freeradiusv1.ClusterSpec{
			Replicas: 1,
			Service:  freeradiusv1.ServiceSpec{Type: "ClusterIP"},
			Certificate: freeradiusv1.CertificateSpec{
				TLSSecretRef: &freeradiusv1.SecretRef{Name: "tls"},
			},
		},
	}
	layout := certs.Layout{
		Mode:      certs.ModeCertManager,
		TLSSecret: freeradiusv1.SecretRef{Name: "tls"},
		TLSHasCA:  true,
	}
	bundle := radiusconfig.Bundle{
		Config: map[string]string{"clients.conf": "a", "authorize": "b"},
		Mods:   map[string]string{"mschap": "m", "eap": "e"},
	}
	out1, err := BuildCluster(desiredClusterInput(cluster, bundle, layout, &certs.Material{CAPEM: []byte("ca")}))
	if err != nil {
		t.Fatal(err)
	}
	bundle2 := radiusconfig.Bundle{
		Config: map[string]string{"authorize": "b", "clients.conf": "a"},
		Mods:   map[string]string{"eap": "e", "mschap": "m"},
	}
	out2, err := BuildCluster(desiredClusterInput(cluster, bundle2, layout, &certs.Material{CAPEM: []byte("ca")}))
	if err != nil {
		t.Fatal(err)
	}
	if out1.ConfigHash != out2.ConfigHash {
		t.Fatalf("expected stable hash, got %q and %q", out1.ConfigHash, out2.ConfigHash)
	}
}

func desiredClusterInput(cluster *freeradiusv1.Cluster, bundle radiusconfig.Bundle, layout certs.Layout, mat *certs.Material) ClusterInput {
	return ClusterInput{
		Cluster:      cluster,
		Bundle:       bundle,
		CertLayout:   layout,
		CertMaterial: mat,
	}
}
