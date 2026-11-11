package ldap

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	freeradiusv1 "github.com/jValdron/freeradius-operator/api/v1"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

const defaultUserFilter = "(uid=%{%{Stripped-User-Name}:-%{User-Name}})"

// Resolve reads secrets and normalizes Cluster LDAP settings for config generation.
func Resolve(ctx context.Context, c client.Client, clusterNS string, spec *freeradiusv1.LDAPSpec) (*radiusconfig.LDAPConfig, error) {
	if spec == nil {
		return nil, nil
	}
	if err := validateServers(spec.Servers, spec.Port); err != nil {
		return nil, err
	}
	if spec.BaseDN == "" {
		return nil, fmt.Errorf("ldap.baseDN is required")
	}
	if spec.FromSecretRef == nil || spec.FromSecretRef.Name == "" {
		return nil, fmt.Errorf("ldap.fromSecretRef is required")
	}

	password, err := readKey(ctx, c, clusterNS, spec.FromSecretRef, "password")
	if err != nil {
		return nil, fmt.Errorf("ldap.fromSecretRef: %w", err)
	}

	identity := strings.TrimSpace(spec.Identity)
	if identity == "" {
		identity, err = readKey(ctx, c, clusterNS, spec.FromSecretRef, "identity")
		if err != nil {
			return nil, fmt.Errorf("ldap identity: set spec.ldap.identity or secret key %q: %w", "identity", err)
		}
	}

	userBase := strings.TrimSpace(spec.UserBaseDN)
	if userBase == "" {
		userBase = spec.BaseDN
	}

	userFilter := wrapAccessGroup(userFilterWithDefault(spec.UserFilter), spec.AccessGroupDN)

	authMode := spec.AuthMode
	if authMode == "" {
		authMode = freeradiusv1.LDAPAuthModeBindAsUser
	}

	ntAttr := strings.TrimSpace(spec.NTPasswordAttribute)
	if ntAttr == "" {
		ntAttr = "ntPassword"
	}

	serverLine, port, err := radiusconfig.FormatLDAPServers(spec.Servers, spec.Port)
	if err != nil {
		return nil, err
	}

	ldaps := false
	for _, s := range spec.Servers {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "ldaps://") {
			ldaps = true
			break
		}
	}

	mountCA := spec.TLS != nil && spec.TLS.CASecretRef != nil && spec.TLS.CASecretRef.Name != ""

	return &radiusconfig.LDAPConfig{
		ServerLine:          serverLine,
		Port:                port,
		BaseDN:              spec.BaseDN,
		UserBaseDN:          userBase,
		Identity:            identity,
		Password:            password,
		UserFilter:          userFilter,
		AuthMode:            authMode,
		NTPasswordAttribute: ntAttr,
		StartTLS:            spec.StartTLS,
		LDAPS:               ldaps,
		MountLdapCA:         mountCA,
		CASecret:            caSecretRef(spec.TLS),
	}, nil
}

func caSecretRef(tls *freeradiusv1.LDAPTLSSpec) freeradiusv1.SecretRef {
	if tls == nil || tls.CASecretRef == nil {
		return freeradiusv1.SecretRef{}
	}
	return *tls.CASecretRef
}

func validateServers(servers []string, port *int32) error {
	if len(servers) == 0 {
		return fmt.Errorf("ldap.servers must contain at least one host")
	}
	if port == nil {
		return nil
	}
	for _, s := range servers {
		if strings.Contains(strings.TrimSpace(s), "://") {
			return fmt.Errorf("ldap.port must not be set when any server is an ldap(s):// URI")
		}
	}
	return nil
}

func userFilterWithDefault(filter string) string {
	f := strings.TrimSpace(filter)
	if f == "" {
		return defaultUserFilter
	}
	return f
}

func wrapAccessGroup(filter, groupDN string) string {
	groupDN = strings.TrimSpace(groupDN)
	if groupDN == "" {
		return filter
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(filter, "("), ")")
	return fmt.Sprintf("(&%s(memberOf=%s))", inner, groupDN)
}

func readKey(ctx context.Context, c client.Client, clusterNS string, ref *freeradiusv1.SecretRef, key string) (string, error) {
	ns := clusterNS
	if ref.Namespace != "" {
		ns = ref.Namespace
	}
	var sec corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: ref.Name}, &sec); err != nil {
		return "", err
	}
	b := sec.Data[key]
	if len(b) == 0 {
		return "", fmt.Errorf("secret %s/%s missing key %q", ns, ref.Name, key)
	}
	return string(b), nil
}
