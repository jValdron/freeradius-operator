package radiusconfig

import freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"

// MergeEAP returns EAP settings with secure defaults applied.
func MergeEAP(spec *freeradiusv1.EAPSpec) freeradiusv1.EAPSpec {
	out := freeradiusv1.EAPSpec{
		DefaultEapType:     "ttls",
		TLSMinVersion:      "1.2",
		TLSMaxVersion:      "1.3",
		CipherList:         "DEFAULT",
		InnerVirtualServer: "inner-tunnel",
	}
	if spec == nil {
		return out
	}
	if spec.DefaultEapType != "" {
		out.DefaultEapType = spec.DefaultEapType
	}
	if spec.TLSMinVersion != "" {
		out.TLSMinVersion = spec.TLSMinVersion
	}
	if spec.TLSMaxVersion != "" {
		out.TLSMaxVersion = spec.TLSMaxVersion
	}
	if spec.CipherList != "" {
		out.CipherList = spec.CipherList
	}
	if spec.InnerVirtualServer != "" {
		out.InnerVirtualServer = spec.InnerVirtualServer
	}
	return out
}
