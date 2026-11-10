package radiusconfig_test

import (
	"strings"
	"testing"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

func TestBuildBundleAuthorizeDeviceMACAndVLAN(t *testing.T) {
	b := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName: "primary",
		Devices: []radiusconfig.DeviceEntry{{
			Name:         "phone",
			MACAddresses: []string{"aa:bb:cc:dd:ee:ff"},
			VLAN:         40,
		}},
	})
	auth := b.Config["authorize"]
	if !strings.Contains(auth, "aabbccddeeff") {
		t.Fatalf("expected normalized MAC in authorize, got:\n%s", auth)
	}
	if !strings.Contains(auth, "Tunnel-Private-Group-Id = 40") {
		t.Fatalf("expected VLAN 40 in authorize, got:\n%s", auth)
	}
}

func TestBuildBundleEAPUsesServerKey(t *testing.T) {
	b := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName: "primary",
		EAP:         &freeradiusv1.EAPSpec{TLSMinVersion: "1.2"},
	})
	eap := b.Mods["eap"]
	if !strings.Contains(eap, "private_key_file = ${certdir}/server.key") {
		t.Fatalf("expected server.key path in eap:\n%s", eap)
	}
	if strings.Contains(eap, "with_ntdomain_hack") {
		t.Fatal("eap should not contain legacy directives")
	}
	if strings.Contains(eap, "dh_file") {
		t.Fatal("expected no dh_file without MountDHFile")
	}
}

func TestBuildBundleEAPWithDHFile(t *testing.T) {
	b := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName: "primary",
		MountDHFile: true,
	})
	if !strings.Contains(b.Mods["eap"], "dh_file = ${certdir}/dh") {
		t.Fatal("expected dh_file when MountDHFile is true")
	}
}

func TestBuildBundleClientsSecretQuoted(t *testing.T) {
	b := radiusconfig.BuildBundle(radiusconfig.Input{
		ClusterName: "primary",
		Clients: []radiusconfig.ClientEntry{{
			Name:      "ap",
			IPAddress: "10.0.0.1",
			Secret:    `foo"bar`,
		}},
	})
	clients := b.Config["clients.conf"]
	if !strings.Contains(clients, `secret = "foo\"bar"`) {
		t.Fatalf("expected escaped secret, got:\n%s", clients)
	}
}

func TestBuildBundleRadiusdNameOnly(t *testing.T) {
	b := radiusconfig.BuildBundle(radiusconfig.Input{ClusterName: "primary"})
	if got := b.Config["radiusd.conf"]; got != "name = primary\n" {
		t.Fatalf("unexpected radiusd.conf: %q", got)
	}
}
