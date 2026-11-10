package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

func readSecretKey(ctx context.Context, c client.Client, ns, name, key string) (string, error) {
	var sec corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &sec); err != nil {
		return "", err
	}
	b := sec.Data[key]
	if len(b) == 0 {
		return "", fmt.Errorf("secret %s/%s missing key %q", ns, name, key)
	}
	return string(b), nil
}

func (r *ClusterReconciler) readSecretString(ctx context.Context, clusterNS string, ref *freeradiusv1.SecretRef, key string) (string, error) {
	ns := clusterNS
	if ref.Namespace != "" {
		ns = ref.Namespace
	}
	return readSecretKey(ctx, r.Client, ns, ref.Name, key)
}
