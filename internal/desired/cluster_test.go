package desired_test

import (
	"strings"
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/desired"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestBuildClusterConfigHashOnDeployment(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		Name: "primary", Namespace: "default",
		Spec: freeradiusv1.ClusterSpec{
			Replicas: 1,
			Service:  freeradiusv1.ServiceSpec{Type: "ClusterIP"},
			Certificate: freeradiusv1.CertificateSpec{
				Certificates: &freeradiusv1.InlineCertificates{
					CA: "ca", PrivateKey: "k", PublicKey: "c",
				},
			},
		},
	}
	mat := &certs.Material{CAPEM: []byte("ca"), ServerKey: []byte("k"), ServerPEM: []byte("c")}
	out, err := desired.BuildCluster(desired.ClusterInput{
		Cluster:      cluster,
		Bundle:       radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "primary"}),
		CertLayout:   certs.Layout{Mode: certs.ModeInline, Material: mat},
		CertMaterial: mat,
	})
	if err != nil {
		t.Fatal(err)
	}
	ann := out.Deployment.Spec.Template.Annotations[desired.ConfigHashAnnotation]
	if ann == "" {
		t.Fatal("expected config-hash annotation")
	}
	if out.ConfigHash != ann {
		t.Fatalf("hash mismatch %q vs %q", out.ConfigHash, ann)
	}
}

func TestBuildClusterCertManagerUniqueCAPMount(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		Name: "primary", Namespace: "default",
		Spec: freeradiusv1.ClusterSpec{
			Replicas: 1,
			Service:  freeradiusv1.ServiceSpec{Type: "ClusterIP"},
			Certificate: freeradiusv1.CertificateSpec{
				TLSSecretRef: &freeradiusv1.SecretRef{Name: "freeradius-primary-tls"},
			},
		},
	}
	mat := &certs.Material{CAPEM: []byte("ca-chain")}
	out, err := desired.BuildCluster(desired.ClusterInput{
		Cluster:      cluster,
		Bundle:       radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "primary"}),
		CertLayout:   certs.Layout{Mode: certs.ModeCertManager, TLSSecret: freeradiusv1.SecretRef{Name: "freeradius-primary-tls"}, TLSHasCA: true},
		CertMaterial: mat,
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, m := range out.Deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		seen[m.MountPath]++
	}
	if seen["/etc/raddb/certs/ca.pem"] != 1 {
		t.Fatalf("expected one ca.pem mount, got %d mounts: %#v", seen["/etc/raddb/certs/ca.pem"], out.Deployment.Spec.Template.Spec.Containers[0].VolumeMounts)
	}
}

func TestBuildClusterNoDebugByDefault(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		Name: "primary", Namespace: "default",
		Spec: freeradiusv1.ClusterSpec{
			Replicas: 1,
			Service:  freeradiusv1.ServiceSpec{Type: "ClusterIP"},
			Certificate: freeradiusv1.CertificateSpec{
				Certificates: &freeradiusv1.InlineCertificates{CA: "ca", PrivateKey: "k", PublicKey: "c"},
			},
		},
	}
	mat := &certs.Material{CAPEM: []byte("ca"), ServerKey: []byte("k"), ServerPEM: []byte("c")}
	out, err := desired.BuildCluster(desired.ClusterInput{
		Cluster:      cluster,
		Bundle:       radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "primary"}),
		CertLayout:   certs.Layout{Mode: certs.ModeInline, Material: mat},
		CertMaterial: mat,
	})
	if err != nil {
		t.Fatal(err)
	}
	args := out.Deployment.Spec.Template.Spec.Containers[0].Args
	want := []string{"-f", "-l", "/dev/stdout"}
	if len(args) != len(want) {
		t.Fatalf("expected radiusd args %v, got %v", want, args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("expected radiusd args %v, got %v", want, args)
		}
	}
	if !strings.Contains(out.Deployment.Spec.Template.Spec.Containers[0].Image, "freeradius") {
		t.Fatalf("unexpected image %s", out.Deployment.Spec.Template.Spec.Containers[0].Image)
	}
}
