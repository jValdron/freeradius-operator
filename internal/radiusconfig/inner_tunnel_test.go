package radiusconfig_test

import (
	"strings"
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestBuildInnerTunnelBindAsUser(t *testing.T) {
	cfg := radiusconfig.LDAPConfig{AuthMode: freeradiusv1.LDAPAuthModeBindAsUser}
	out := radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "x", LDAP: &cfg})
	site := out.Config["inner-tunnel"]
	if strings.Contains(site, "\t-ldap") {
		t.Fatalf("expected ldap enabled, still has -ldap")
	}
	if !strings.Contains(site, "control:Auth-Type := ldap") {
		t.Fatalf("missing bind-as-user stanza")
	}
	if !strings.Contains(site, "\tAuth-Type LDAP {") {
		t.Fatalf("Auth-Type LDAP block not enabled")
	}
}

func TestBuildInnerTunnelMschap(t *testing.T) {
	cfg := radiusconfig.LDAPConfig{AuthMode: freeradiusv1.LDAPAuthModeMSCHAP}
	out := radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "x", LDAP: &cfg})
	site := out.Config["inner-tunnel"]
	if strings.Contains(site, "control:Auth-Type := ldap") {
		t.Fatalf("mschap should not use bind-as-user authorize stanza")
	}
	if strings.Contains(site, "\t-ldap") {
		t.Fatalf("expected ldap enabled")
	}
}
