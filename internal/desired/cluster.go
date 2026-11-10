package desired

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/labels"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

const defaultFreeradiusImage = "freeradius/freeradius-server:3.2.10-alpine"

const metallbAllowSharedIP = "metallb.universe.tf/allow-shared-ip"

// ClusterInput is the inputs required to render child resources for one Cluster.
type ClusterInput struct {
	Cluster    *freeradiusv1.Cluster
	Bundle     radiusconfig.Bundle
	CertLayout certs.Layout
	// Material populated after resolving legacy/inline/dh for cert-manager+inline dh.
	CertMaterial *certs.Material
}

// ClusterOutput contains desired child objects for a cluster reconcile.
type ClusterOutput struct {
	ConfigMap   *corev1.ConfigMap
	ModsMap     *corev1.ConfigMap
	CertsSecret *corev1.Secret
	Deployment  *appsv1.Deployment
	Service     *corev1.Service
	ConfigHash  string
}

// BuildCluster assembles ConfigMaps, Secret, Deployment, and Service.
func BuildCluster(in ClusterInput) (ClusterOutput, error) {
	cluster := in.Cluster
	if cluster == nil {
		return ClusterOutput{}, fmt.Errorf("cluster is required")
	}
	name := cluster.Name
	ns := cluster.Namespace
	lbls := labels.ForCluster(name)

	configHash := hashConfig(in.Bundle, in.CertLayout, in.CertMaterial, cluster.Spec)

	configCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapConfigName(name),
			Namespace: ns,
			Labels:    lbls,
		},
		Data: in.Bundle.Config,
	}
	modsCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapModsName(name),
			Namespace: ns,
			Labels:    lbls,
		},
		Data: in.Bundle.Mods,
	}

	var certSecret *corev1.Secret
	if mat := effectiveMaterial(in.CertLayout, in.CertMaterial); mat != nil {
		certSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certsSecretName(name),
				Namespace: ns,
				Labels:    lbls,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"ca.pem":     mat.CAPEM,
				"dh":         mat.DH,
				"server.key": mat.ServerKey,
				"server.pem": mat.ServerPEM,
			},
		}
	}

	image := cluster.Spec.Image
	if image == "" {
		image = defaultFreeradiusImage
	}

	mountDH := certs.MountDHParams(cluster.Spec.Certificate, in.CertLayout, in.CertMaterial)
	hasCertSecret := certSecret != nil

	replicas := cluster.Spec.Replicas
	if replicas < 0 {
		replicas = 1
	}

	podLabels := map[string]string{
		labels.AppLabel:   labels.AppFreeradius,
		labels.ClusterKey: name,
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName(name),
			Namespace: ns,
			Labels:    lbls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
					Annotations: map[string]string{
						ConfigHashAnnotation: configHash,
					},
				},
				Spec: corev1.PodSpec{
					Volumes:            buildVolumes(name, in.CertLayout, hasCertSecret, mountDH),
					Containers:         []corev1.Container{buildContainer(image, cluster.Spec.Debug, name, in.CertLayout, hasCertSecret, mountDH)},
					ServiceAccountName: "default",
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName(name),
			Namespace: ns,
			Labels:    lbls,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(cluster.Spec.Service.Type),
			Selector: podLabels,
			Ports: []corev1.ServicePort{
				{Name: "auth", Port: 1812, Protocol: corev1.ProtocolUDP, TargetPort: intstr.FromInt32(1812)},
				{Name: "acct", Port: 1813, Protocol: corev1.ProtocolUDP, TargetPort: intstr.FromInt32(1813)},
			},
		},
	}
	if cluster.Spec.Service.LoadBalancerIP != "" {
		svc.Spec.LoadBalancerIP = cluster.Spec.Service.LoadBalancerIP
		if cluster.Spec.Service.Type == string(corev1.ServiceTypeLoadBalancer) {
			if svc.Annotations == nil {
				svc.Annotations = map[string]string{}
			}
			svc.Annotations[metallbAllowSharedIP] = "true"
		}
	}

	return ClusterOutput{
		ConfigMap:   configCM,
		ModsMap:     modsCM,
		CertsSecret: certSecret,
		Deployment:  dep,
		Service:     svc,
		ConfigHash:  configHash,
	}, nil
}

func effectiveMaterial(layout certs.Layout, resolved *certs.Material) *certs.Material {
	switch layout.Mode {
	case certs.ModeInline:
		return layout.Material
	case certs.ModeLegacySecret:
		return resolved
	case certs.ModeCertManager:
		// TLS cert/key/CA mount from the cert-manager Secret; generated Secret is only for optional DH.
		if resolved == nil || len(resolved.DH) == 0 {
			return nil
		}
		return &certs.Material{DH: resolved.DH}
	default:
		return nil
	}
}

func hashConfig(bundle radiusconfig.Bundle, layout certs.Layout, mat *certs.Material, spec freeradiusv1.ClusterSpec) string {
	h := sha256.New()
	keys := make([]string, 0, len(bundle.Config))
	for k := range bundle.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(bundle.Config[k]))
	}
	for k, v := range bundle.Mods {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(v))
	}
	if mat != nil {
		_, _ = h.Write(mat.CAPEM)
		_, _ = h.Write(mat.DH)
		_, _ = h.Write(mat.ServerKey)
		_, _ = h.Write(mat.ServerPEM)
	}
	_, _ = fmt.Fprintf(h, "cert=%d", layout.Mode)
	if layout.TLSSecret.Name != "" {
		_, _ = h.Write([]byte(layout.TLSSecret.Name))
		_, _ = h.Write([]byte(layout.TLSSecret.Namespace))
	}
	_, _ = fmt.Fprintf(h, "image=%s", spec.Image)
	_, _ = fmt.Fprintf(h, "debug=%v", spec.Debug)
	_, _ = fmt.Fprintf(h, "replicas=%d", spec.Replicas)
	return hex.EncodeToString(h.Sum(nil))
}
