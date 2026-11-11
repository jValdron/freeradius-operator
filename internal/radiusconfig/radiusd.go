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

const (
	defaultRadiusdNameLine    = "name = radiusd"
	defaultLogDestinationLine = "destination = files"
	k8sLogDestinationLine     = "destination = stdout"
)

func buildRadiusd(clusterName string) string {
	if clusterName == "" {
		clusterName = "radiusd"
	}
	conf := string(radiusdConfDefault)
	nameLine := fmt.Sprintf("name = %s", clusterName)
	conf = strings.Replace(conf, defaultRadiusdNameLine, nameLine, 1)
	conf = strings.Replace(conf, defaultLogDestinationLine, k8sLogDestinationLine, 1)
	return conf
}
