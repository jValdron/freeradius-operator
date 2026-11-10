package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/config"
	"github.com/jValdron/freeradius-operator/internal/desired"
	"github.com/jValdron/freeradius-operator/internal/ensure"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

// ClusterReconciler reconciles a Cluster object.
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.Config

	gate reconcileGate
}

// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=clients,verbs=get;list;watch
// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=users,verbs=get;list;watch
// +kubebuilder:rbac:groups=freeradius.valdron.ca,resources=devices,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.gate.run(req.NamespacedName.String(), func() (ctrl.Result, error) {
		return r.reconcile(ctx, req)
	})
}

func (r *ClusterReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var cluster freeradiusv1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var all freeradiusv1.ClusterList
	if err := r.List(ctx, &all, client.InNamespace(cluster.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	agg, err := r.aggregateForCluster(ctx, &cluster, all.Items)
	if err != nil {
		setCondition(&cluster, freeradiusv1.ConditionReady, metav1.ConditionFalse, "AggregateFailed", err.Error())
		_ = r.patchStatus(ctx, &cluster)
		return ctrl.Result{}, err
	}

	layout, err := certs.LayoutFromCluster(cluster.Spec.Certificate)
	if err != nil {
		setCondition(&cluster, freeradiusv1.ConditionCertificateConfigured, metav1.ConditionFalse, "InvalidCertificate", err.Error())
		_ = r.patchStatus(ctx, &cluster)
		return ctrl.Result{}, err
	}

	mat, err := certs.ResolveMaterial(ctx, r.Client, cluster.Namespace, cluster.Spec.Certificate)
	if err != nil {
		setCondition(&cluster, freeradiusv1.ConditionCertificateConfigured, metav1.ConditionFalse, "CertificateResolveFailed", err.Error())
		_ = r.patchStatus(ctx, &cluster)
		return ctrl.Result{}, err
	}
	setCondition(&cluster, freeradiusv1.ConditionCertificateConfigured, metav1.ConditionTrue, "Resolved", "certificate material resolved")

	password := cluster.Spec.Certificate.PrivateKeyPassword
	if mat != nil && mat.KeyPassword != "" {
		password = mat.KeyPassword
	}

	bundle := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName:        cluster.Name,
		DefaultVlan:        cluster.Spec.DefaultVlan,
		EAP:                cluster.Spec.EAP,
		PrivateKeyPassword: password,
		MountDHFile:        certs.MountDHParams(cluster.Spec.Certificate, layout, mat),
		Users:              agg.Users,
		Devices:            agg.Devices,
		Clients:            agg.Clients,
	})

	out, err := desired.BuildCluster(desired.ClusterInput{
		Cluster:      &cluster,
		Bundle:       bundle,
		CertLayout:   layout,
		CertMaterial: mat,
	})
	if err != nil {
		setCondition(&cluster, freeradiusv1.ConditionReady, metav1.ConditionFalse, "BuildFailed", err.Error())
		_ = r.patchStatus(ctx, &cluster)
		return ctrl.Result{}, err
	}

	owner := cluster.DeepCopy()
	if err := r.applyConfigMap(ctx, out.ConfigMap, owner); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.applyConfigMap(ctx, out.ModsMap, owner); err != nil {
		return ctrl.Result{}, err
	}
	if out.CertsSecret != nil {
		if err := r.applySecret(ctx, out.CertsSecret, owner); err != nil {
			return ctrl.Result{}, err
		}
	}
	if err := r.applyDeployment(ctx, out.Deployment, owner); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.applyService(ctx, out.Service, owner); err != nil {
		return ctrl.Result{}, err
	}

	setCondition(&cluster, freeradiusv1.ConditionWorkloadReady, metav1.ConditionTrue, "Applied", "generated resources updated")
	ready := apimeta.IsStatusConditionTrue(cluster.Status.Conditions, freeradiusv1.ConditionCertificateConfigured) &&
		apimeta.IsStatusConditionTrue(cluster.Status.Conditions, freeradiusv1.ConditionWorkloadReady)
	if ready {
		setCondition(&cluster, freeradiusv1.ConditionReady, metav1.ConditionTrue, "Ready", "cluster reconciled")
	}
	cluster.Status.ObservedGeneration = cluster.Generation
	if err := r.patchStatus(ctx, &cluster); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled cluster", "namespace", cluster.Namespace, "name", cluster.Name, "configHash", out.ConfigHash)
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) applyConfigMap(ctx context.Context, desired *corev1.ConfigMap, owner client.Object) error {
	live := &corev1.ConfigMap{}
	live.Name = desired.Name
	live.Namespace = desired.Namespace
	_, err := ensure.Apply(ctx, r.Client, live, owner, r.Config.DryRun, func() error {
		live.Data = desired.Data
		live.Labels = desired.Labels
		return nil
	})
	return err
}

func (r *ClusterReconciler) applySecret(ctx context.Context, desired *corev1.Secret, owner client.Object) error {
	live := &corev1.Secret{}
	live.Name = desired.Name
	live.Namespace = desired.Namespace
	_, err := ensure.Apply(ctx, r.Client, live, owner, r.Config.DryRun, func() error {
		live.Data = desired.Data
		live.Labels = desired.Labels
		live.Type = desired.Type
		return nil
	})
	return err
}

func (r *ClusterReconciler) applyDeployment(ctx context.Context, desired *appsv1.Deployment, owner client.Object) error {
	live := &appsv1.Deployment{}
	live.Name = desired.Name
	live.Namespace = desired.Namespace
	_, err := ensure.Apply(ctx, r.Client, live, owner, r.Config.DryRun, func() error {
		live.Spec = desired.Spec
		live.Labels = desired.Labels
		return nil
	})
	return err
}

func (r *ClusterReconciler) applyService(ctx context.Context, desired *corev1.Service, owner client.Object) error {
	live := &corev1.Service{}
	live.Name = desired.Name
	live.Namespace = desired.Namespace
	_, err := ensure.Apply(ctx, r.Client, live, owner, r.Config.DryRun, func() error {
		live.Spec = desired.Spec
		live.Labels = desired.Labels
		live.Annotations = desired.Annotations
		return nil
	})
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&freeradiusv1.Cluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Watches(&freeradiusv1.Client{}, handler.EnqueueRequestsFromMapFunc(r.mapToClusters)).
		Watches(&freeradiusv1.User{}, handler.EnqueueRequestsFromMapFunc(r.mapToClusters)).
		Watches(&freeradiusv1.Device{}, handler.EnqueueRequestsFromMapFunc(r.mapToClusters)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToClusters)).
		Named("cluster").
		Complete(r)
}

func (r *ClusterReconciler) mapToClusters(ctx context.Context, obj client.Object) []ctrl.Request {
	list := &freeradiusv1.ClusterList{}
	if err := r.List(ctx, list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	reqs := make([]ctrl.Request, 0, len(list.Items))
	for i := range list.Items {
		reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return reqs
}

func (r *ClusterReconciler) mapSecretToClusters(ctx context.Context, obj client.Object) []ctrl.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	list := &freeradiusv1.ClusterList{}
	if err := r.List(ctx, list, client.InNamespace(secret.Namespace)); err != nil {
		return nil
	}
	var reqs []ctrl.Request
	for i := range list.Items {
		c := &list.Items[i]
		if clusterReferencesSecret(c, secret.Namespace, secret.Name) {
			reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(c)})
		}
	}
	return reqs
}

func clusterReferencesSecret(c *freeradiusv1.Cluster, ns, name string) bool {
	spec := c.Spec.Certificate
	refs := []*freeradiusv1.SecretRef{spec.FromSecretRef, spec.TLSSecretRef, spec.DHSecretRef}
	for _, ref := range refs {
		if ref == nil || ref.Name != name {
			continue
		}
		refNS := c.Namespace
		if ref.Namespace != "" {
			refNS = ref.Namespace
		}
		if refNS == ns {
			return true
		}
	}
	return false
}

// compile-time check for generics helper
