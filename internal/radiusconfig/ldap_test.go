package radiusconfig_test

import (
	"strings"
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestFormatLDAPServers(t *testing.T) {
	t.Run("multi host with port", func(t *testing.T) {
		port := int32(389)
		line, p, err := radiusconfig.FormatLDAPServers([]string{"ldap1", "ldap2"}, &port)
		if err != nil {
			t.Fatal(err)
		}
		if line != "ldap1 ldap2" || p == nil || *p != 389 {
			t.Fatalf("got line=%q port=%v", line, p)
		}
	})

	t.Run("reject port with URI", func(t *testing.T) {
		port := int32(389)
		_, _, err := radiusconfig.FormatLDAPServers([]string{"ldaps://ldap:636"}, &port)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBuildLDAPBindAsUser(t *testing.T) {
	cfg := radiusconfig.LDAPConfig{
		ServerLine: "ldap.example.com",
		BaseDN:     "dc=example,dc=com",
		UserBaseDN: "dc=example,dc=com",
		Identity:   "cn=admin,dc=example,dc=com",
		Password:   "secret",
		UserFilter: "(uid=%{User-Name})",
		AuthMode:   freeradiusv1.LDAPAuthModeBindAsUser,
	}
	out := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName: "x",
		LDAP:        &cfg,
	})
	ldapMod := out.Mods["ldap"]
	if !strings.Contains(ldapMod, `server = "ldap.example.com"`) {
		t.Fatalf("missing server: %s", ldapMod)
	}
	if strings.Contains(ldapMod, "NT-Password") {
		t.Fatalf("bind-as-user should not map NT-Password: %s", ldapMod)
	}
}

func TestBuildLDAPMschap(t *testing.T) {
	cfg := radiusconfig.LDAPConfig{
		ServerLine:          "ldap.example.com",
		BaseDN:              "dc=example,dc=com",
		UserBaseDN:          "dc=example,dc=com",
		Identity:            "cn=admin,dc=example,dc=com",
		Password:            "secret",
		UserFilter:          "(uid=x)",
		AuthMode:            freeradiusv1.LDAPAuthModeMSCHAP,
		NTPasswordAttribute: "ipaNTHash",
	}
	out := radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "x", LDAP: &cfg})
	if !strings.Contains(out.Mods["ldap"], `control:NT-Password := "ipaNTHash"`) {
		t.Fatalf("mschap mapping missing: %s", out.Mods["ldap"])
	}
}
