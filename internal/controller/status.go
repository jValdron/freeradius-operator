package controller

import (
	"context"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

func setCondition(cluster *freeradiusv1.Cluster, condType string, status metav1.ConditionStatus, reason, message string) {
	cond := metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cluster.Generation,
		LastTransitionTime: metav1.Now(),
	}
	apimeta.SetStatusCondition(&cluster.Status.Conditions, cond)
}

func (r *ClusterReconciler) patchStatus(ctx context.Context, cluster *freeradiusv1.Cluster) error {
	return r.Status().Update(ctx, cluster)
}
