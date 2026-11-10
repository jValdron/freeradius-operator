package labels

const (
	ManagedBy      = "app.kubernetes.io/managed-by"
	ManagedByValue = "freeradius-operator"

	AppLabel      = "app"
	AppFreeradius = "freeradius"

	// ClusterNameLabel selects Client/User/Device for a non-default cluster (legacy key).
	ClusterNameLabel = "clusterName"

	ClusterKey = "freeradius.valdron.ca/cluster"
)

// ForCluster returns labels applied to generated resources.
func ForCluster(clusterName string) map[string]string {
	return map[string]string{
		ManagedBy:  ManagedByValue,
		AppLabel:   AppFreeradius,
		ClusterKey: clusterName,
	}
}
