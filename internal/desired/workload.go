package desired

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/jValdron/freeradius-operator/internal/certs"
	"github.com/jValdron/freeradius-operator/internal/radiusconfig"
)

const (
	volRaddbConfig = "raddb-config"
	volRaddbMods   = "raddb-mods"
	volRaddbTLS    = "raddb-tls"
	volRaddbDH     = "raddb-dh"
	volRaddbCerts  = "raddb-certs"
	volRaddbLDAPCA = "raddb-ldap-ca"

	mountRaddbCA = "/etc/raddb/certs/ca.pem"
	mountRaddbDH = "/etc/raddb/certs/dh"
)

type ldapVolumeOpts struct {
	Enabled      bool
	Namespace    string
	CASecretName string
	CASecretNS   string
}

func buildVolumes(clusterName string, layout certs.Layout, hasGeneratedCerts bool, ldap ldapVolumeOpts) []corev1.Volume {
	vols := []corev1.Volume{
		{
			Name: volRaddbConfig,
			ConfigMap: &corev1.ConfigMapVolumeSource{
				Name: configMapConfigName(clusterName),
			},
		},
		{
			Name: volRaddbMods,
			ConfigMap: &corev1.ConfigMapVolumeSource{
				Name: configMapModsName(clusterName),
			},
		},
	}

	switch layout.Mode {
	case certs.ModeCertManager:
		vols = append(vols, corev1.Volume{
			Name: volRaddbTLS,
			Secret: &corev1.SecretVolumeSource{
				SecretName: layout.TLSSecret.Name,
			},
		})
		if layout.DHSecret.Name != "" {
			vols = append(vols, corev1.Volume{
				Name: volRaddbDH,
				Secret: &corev1.SecretVolumeSource{
					SecretName: layout.DHSecret.Name,
				},
			})
		} else if hasGeneratedCerts {
			vols = append(vols, corev1.Volume{
				Name: volRaddbCerts,
				Secret: &corev1.SecretVolumeSource{
					SecretName: certsSecretName(clusterName),
				},
			})
		}
	default:
		if hasGeneratedCerts {
			vols = append(vols, corev1.Volume{
				Name: volRaddbCerts,
				Secret: &corev1.SecretVolumeSource{
					SecretName: certsSecretName(clusterName),
				},
			})
		}
	}
	if ldap.Enabled && ldap.CASecretName != "" {
		vols = append(vols, corev1.Volume{
			Name: volRaddbLDAPCA,
			Secret: &corev1.SecretVolumeSource{
				SecretName: ldap.CASecretName,
			},
		})
	}
	return vols
}

func buildContainer(image string, debug bool, layout certs.Layout, hasGeneratedCerts, mountDH bool, ldap ldapVolumeOpts) corev1.Container {
	args := []string{"-f"}
	if debug {
		args = append(args, "-X")
	}
	c := corev1.Container{
		Name:    "freeradius",
		Image:   image,
		Command: []string{"/opt/sbin/radiusd"},
		Args:    args,
		Ports: []corev1.ContainerPort{
			{ContainerPort: 1812, Protocol: corev1.ProtocolUDP, Name: "auth"},
			{ContainerPort: 1813, Protocol: corev1.ProtocolUDP, Name: "acct"},
		},
		VolumeMounts: configVolumeMounts(ldap.Enabled),
	}
	c.VolumeMounts = append(c.VolumeMounts, certVolumeMounts(layout, hasGeneratedCerts, mountDH)...)
	if ldap.Enabled && ldap.CASecretName != "" {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      volRaddbLDAPCA,
			MountPath: radiusconfig.LDAPCAFile,
			SubPath:   "ca.crt",
			ReadOnly:  true,
		})
	}
	return c
}

func configVolumeMounts(ldapEnabled bool) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: volRaddbConfig, MountPath: "/etc/raddb/mods-config/files/authorize", SubPath: "authorize"},
		{Name: volRaddbConfig, MountPath: "/etc/raddb/clients.conf", SubPath: "clients.conf"},
		{Name: volRaddbConfig, MountPath: "/etc/raddb/radiusd.conf", SubPath: "radiusd.conf"},
		{Name: volRaddbMods, MountPath: "/etc/raddb/mods-enabled/eap", SubPath: "eap"},
		{Name: volRaddbMods, MountPath: "/etc/raddb/mods-enabled/mschap", SubPath: "mschap"},
	}
	if ldapEnabled {
		mounts = append(mounts,
			corev1.VolumeMount{Name: volRaddbMods, MountPath: "/etc/raddb/mods-enabled/ldap", SubPath: "ldap"},
			corev1.VolumeMount{Name: volRaddbConfig, MountPath: "/etc/raddb/sites-enabled/inner-tunnel", SubPath: "inner-tunnel"},
		)
	}
	return mounts
}

func certVolumeMounts(layout certs.Layout, hasGeneratedCerts, mountDH bool) []corev1.VolumeMount {
	switch layout.Mode {
	case certs.ModeCertManager:
		mounts := []corev1.VolumeMount{
			{Name: volRaddbTLS, MountPath: "/etc/raddb/certs/server.pem", SubPath: "tls.crt"},
			{Name: volRaddbTLS, MountPath: "/etc/raddb/certs/server.key", SubPath: "tls.key"},
		}
		if layoutHasCA(layout) {
			mounts = append(mounts, corev1.VolumeMount{Name: volRaddbTLS, MountPath: mountRaddbCA, SubPath: "ca.crt"})
		} else {
			mounts = append(mounts, corev1.VolumeMount{Name: volRaddbTLS, MountPath: mountRaddbCA, SubPath: "tls.crt"})
		}
		if mountDH {
			if layout.DHSecret.Name != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: volRaddbDH, MountPath: mountRaddbDH, SubPath: "dh"})
			} else if hasGeneratedCerts {
				mounts = append(mounts, corev1.VolumeMount{Name: volRaddbCerts, MountPath: mountRaddbDH, SubPath: "dh"})
			}
		}
		return mounts
	default:
		mounts := []corev1.VolumeMount{
			{Name: volRaddbCerts, MountPath: mountRaddbCA, SubPath: "ca.pem"},
			{Name: volRaddbCerts, MountPath: "/etc/raddb/certs/server.key", SubPath: "server.key"},
			{Name: volRaddbCerts, MountPath: "/etc/raddb/certs/server.pem", SubPath: "server.pem"},
		}
		if mountDH {
			mounts = append(mounts, corev1.VolumeMount{Name: volRaddbCerts, MountPath: mountRaddbDH, SubPath: "dh"})
		}
		return mounts
	}
}

func layoutHasCA(layout certs.Layout) bool {
	return layout.TLSHasCA
}
