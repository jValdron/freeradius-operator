package radiusconfig

import (
	_ "embed"
	"fmt"
	"strings"
)

// Stock radiusd.conf from freeradius/freeradius-server:3.2.10-alpine (/etc/raddb/radiusd.conf).
// We embed the full file because the ConfigMap subPath replaces the image default entirely.
//
//go:embed embed/radiusd.conf.default
var radiusdConfDefault []byte

const defaultRadiusdNameLine = "name = radiusd"

func buildRadiusd(clusterName string) string {
	if clusterName == "" {
		clusterName = "radiusd"
	}
	conf := string(radiusdConfDefault)
	nameLine := fmt.Sprintf("name = %s", clusterName)
	if strings.Contains(conf, defaultRadiusdNameLine) {
		return strings.Replace(conf, defaultRadiusdNameLine, nameLine, 1)
	}
	return nameLine + "\n" + conf
}
