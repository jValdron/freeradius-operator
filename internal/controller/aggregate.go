package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/labels"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

type aggregated struct {
	Clients []radiusconfig.ClientEntry
	Users   []radiusconfig.UserEntry
	Devices []radiusconfig.DeviceEntry
}

func (r *ClusterReconciler) aggregateForCluster(ctx context.Context, cluster *freeradiusv1.Cluster, allClusters []freeradiusv1.Cluster) (aggregated, error) {
	isDefault := clusterDefault(cluster, allClusters)

	var clients freeradiusv1.ClientList
	if err := r.List(ctx, &clients, client.InNamespace(cluster.Namespace)); err != nil {
		return aggregated{}, err
	}
	var users freeradiusv1.UserList
	if err := r.List(ctx, &users, client.InNamespace(cluster.Namespace)); err != nil {
		return aggregated{}, err
	}
	var devices freeradiusv1.DeviceList
	if err := r.List(ctx, &devices, client.InNamespace(cluster.Namespace)); err != nil {
		return aggregated{}, err
	}

	out := aggregated{}
	for i := range clients.Items {
		c := &clients.Items[i]
		if !resourceBelongsToCluster(c.Labels, cluster.Name, isDefault) {
			continue
		}
		secret := c.Spec.Secret
		if c.Spec.FromSecretRef != nil {
			s, err := r.readSecretString(ctx, cluster.Namespace, c.Spec.FromSecretRef, "secret")
			if err != nil {
				return aggregated{}, fmt.Errorf("client %s: %w", c.Name, err)
			}
			secret = s
		}
		out.Clients = append(out.Clients, radiusconfig.ClientEntry{
			Name:      c.Name,
			IPAddress: c.Spec.IPAddress,
			Secret:    secret,
		})
	}
	for i := range users.Items {
		u := &users.Items[i]
		if !resourceBelongsToCluster(u.Labels, cluster.Name, isDefault) {
			continue
		}
		username, password := u.Spec.Username, u.Spec.Password
		if u.Spec.FromSecretRef != nil {
			var err error
			username, err = r.readSecretString(ctx, cluster.Namespace, u.Spec.FromSecretRef, "username")
			if err != nil {
				return aggregated{}, fmt.Errorf("user %s: %w", u.Name, err)
			}
			password, err = r.readSecretString(ctx, cluster.Namespace, u.Spec.FromSecretRef, "password")
			if err != nil {
				return aggregated{}, fmt.Errorf("user %s: %w", u.Name, err)
			}
		}
		out.Users = append(out.Users, radiusconfig.UserEntry{
			Name:     u.Name,
			Username: username,
			Password: password,
		})
	}
	for i := range devices.Items {
		d := &devices.Items[i]
		if !resourceBelongsToCluster(d.Labels, cluster.Name, isDefault) {
			continue
		}
		out.Devices = append(out.Devices, radiusconfig.DeviceEntry{
			Name:         d.Name,
			MACAddresses: d.Spec.MACAddresses,
			VLAN:         d.Spec.VLAN,
		})
	}
	return out, nil
}

func clusterDefault(cluster *freeradiusv1.Cluster, all []freeradiusv1.Cluster) bool {
	if cluster.Spec.IsDefaultCluster != nil {
		return *cluster.Spec.IsDefaultCluster
	}
	return len(all) == 1
}

func resourceBelongsToCluster(lbls map[string]string, clusterName string, isDefault bool) bool {
	if lbls == nil {
		return isDefault
	}
	if v, ok := lbls[labels.ClusterNameLabel]; ok {
		return v == clusterName
	}
	if v, ok := lbls[labels.ClusterKey]; ok {
		return v == clusterName
	}
	return isDefault
}
