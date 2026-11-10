package radiusconfig

import "fmt"

// buildRadiusd overrides only the instance name; other directives use the container image defaults.
func buildRadiusd(clusterName string) string {
	return fmt.Sprintf("name = %s\n", clusterName)
}
