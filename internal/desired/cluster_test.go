package desired_test

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/desired"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestBuildClusterConfigHashOnDeployment(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "primary", Namespace: "default"},
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

func TestBuildClusterNoDebugByDefault(t *testing.T) {
	cluster := &freeradiusv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "primary", Namespace: "default"},
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
	for _, a := range out.Deployment.Spec.Template.Spec.Containers[0].Args {
		if a == "-X" {
			t.Fatal("expected no -X without debug")
		}
	}
	if !strings.Contains(out.Deployment.Spec.Template.Spec.Containers[0].Image, "freeradius") {
		t.Fatalf("unexpected image %s", out.Deployment.Spec.Template.Spec.Containers[0].Image)
	}
}
