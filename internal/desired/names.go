package desired

import "fmt"

const (
	ConfigHashAnnotation = "freeradius.valdron.ca/config-hash"
)

func deploymentName(clusterName string) string {
	return fmt.Sprintf("freeradius-%s", clusterName)
}

func serviceName(clusterName string) string {
	return deploymentName(clusterName)
}

func configMapConfigName(clusterName string) string {
	return fmt.Sprintf("freeradius-%s-config-generated", clusterName)
}

func configMapModsName(clusterName string) string {
	return fmt.Sprintf("freeradius-%s-mods-generated", clusterName)
}

func certsSecretName(clusterName string) string {
	return fmt.Sprintf("freeradius-%s-certs-generated", clusterName)
}
