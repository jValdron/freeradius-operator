package radiusconfig

import (
	"fmt"
	"strings"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
)

func buildEAP(cfg freeradiusv1.EAPSpec, privateKeyPassword string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "eap {\n")
	fmt.Fprintf(&b, "\tdefault_eap_type = %s\n\n", cfg.DefaultEapType)

	fmt.Fprintf(&b, "\ttls-config tls-common {\n")
	if privateKeyPassword != "" {
		fmt.Fprintf(&b, "\t\tprivate_key_password = %q\n", privateKeyPassword)
	}
	fmt.Fprintf(&b, "\t\tprivate_key_file = ${certdir}/server.key\n")
	fmt.Fprintf(&b, "\t\tcertificate_file = ${certdir}/server.pem\n")
	fmt.Fprintf(&b, "\t\tca_file = ${cadir}/ca.pem\n")
	fmt.Fprintf(&b, "\t\tdh_file = ${certdir}/dh\n")
	fmt.Fprintf(&b, "\t\tcipher_list = %q\n", cfg.CipherList)
	fmt.Fprintf(&b, "\t\ttls_min_version = %q\n", cfg.TLSMinVersion)
	fmt.Fprintf(&b, "\t\ttls_max_version = %q\n", cfg.TLSMaxVersion)
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\ttls {\n\t\ttls = tls-common\n\t}\n\n")

	fmt.Fprintf(&b, "\tgtc {\n\t\tauth_type = PAP\n\t}\n\n")

	fmt.Fprintf(&b, "\tttls {\n")
	fmt.Fprintf(&b, "\t\ttls = tls-common\n")
	fmt.Fprintf(&b, "\t\tdefault_eap_type = gtc\n")
	fmt.Fprintf(&b, "\t\tcopy_request_to_tunnel = no\n")
	fmt.Fprintf(&b, "\t\tuse_tunneled_reply = no\n")
	fmt.Fprintf(&b, "\t\tvirtual_server = %q\n", cfg.InnerVirtualServer)
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tpeap {\n")
	fmt.Fprintf(&b, "\t\ttls = tls-common\n")
	fmt.Fprintf(&b, "\t\tdefault_eap_type = gtc\n")
	fmt.Fprintf(&b, "\t\tcopy_request_to_tunnel = no\n")
	fmt.Fprintf(&b, "\t\tuse_tunneled_reply = no\n")
	fmt.Fprintf(&b, "\t\tvirtual_server = %q\n", cfg.InnerVirtualServer)
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tmschapv2 {}\n")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}
