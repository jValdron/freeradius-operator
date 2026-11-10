package desired

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/jValdron/freeradius-operator/internal/certs"
)

func buildVolumes(clusterName string, layout certs.Layout, hasGeneratedCerts, mountDH bool) []corev1.Volume {
	vols := []corev1.Volume{
		{
			Name: "raddb-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapConfigName(clusterName)},
				},
			},
		},
		{
			Name: "raddb-mods",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapModsName(clusterName)},
				},
			},
		},
	}

	switch layout.Mode {
	case certs.ModeCertManager:
		vols = append(vols, corev1.Volume{
			Name: "raddb-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: layout.TLSSecret.Name,
				},
			},
		})
		if layout.DHSecret.Name != "" {
			vols = append(vols, corev1.Volume{
				Name: "raddb-dh",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: layout.DHSecret.Name,
					},
				},
			})
		} else if hasGeneratedCerts {
			vols = append(vols, corev1.Volume{
				Name: "raddb-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: certsSecretName(clusterName),
					},
				},
			})
		}
	default:
		if hasGeneratedCerts {
			vols = append(vols, corev1.Volume{
				Name: "raddb-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: certsSecretName(clusterName),
					},
				},
			})
		}
	}
	return vols
}

func buildContainer(image string, debug bool, clusterName string, layout certs.Layout, hasGeneratedCerts, mountDH bool) corev1.Container {
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
		VolumeMounts: configVolumeMounts(),
	}
	c.VolumeMounts = append(c.VolumeMounts, certVolumeMounts(layout, clusterName, hasGeneratedCerts, mountDH)...)
	return c
}

func configVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: "raddb-config", MountPath: "/etc/raddb/mods-config/files/authorize", SubPath: "authorize"},
		{Name: "raddb-config", MountPath: "/etc/raddb/clients.conf", SubPath: "clients.conf"},
		{Name: "raddb-config", MountPath: "/etc/raddb/radiusd.conf", SubPath: "radiusd.conf"},
		{Name: "raddb-mods", MountPath: "/etc/raddb/mods-enabled/eap", SubPath: "eap"},
		{Name: "raddb-mods", MountPath: "/etc/raddb/mods-enabled/mschap", SubPath: "mschap"},
	}
}

func certVolumeMounts(layout certs.Layout, clusterName string, hasGeneratedCerts, mountDH bool) []corev1.VolumeMount {
	switch layout.Mode {
	case certs.ModeCertManager:
		mounts := []corev1.VolumeMount{
			{Name: "raddb-tls", MountPath: "/etc/raddb/certs/server.pem", SubPath: "tls.crt"},
			{Name: "raddb-tls", MountPath: "/etc/raddb/certs/server.key", SubPath: "tls.key"},
		}
		if _, ok := layoutHasCA(layout); ok {
			mounts = append(mounts, corev1.VolumeMount{Name: "raddb-tls", MountPath: "/etc/raddb/certs/ca.pem", SubPath: "ca.crt"})
		} else {
			mounts = append(mounts, corev1.VolumeMount{Name: "raddb-tls", MountPath: "/etc/raddb/certs/ca.pem", SubPath: "tls.crt"})
		}
		if mountDH {
			if layout.DHSecret.Name != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: "raddb-dh", MountPath: "/etc/raddb/certs/dh", SubPath: "dh"})
			} else if hasGeneratedCerts {
				mounts = append(mounts, corev1.VolumeMount{Name: "raddb-certs", MountPath: "/etc/raddb/certs/dh", SubPath: "dh"})
			}
		}
		return mounts
	default:
		mounts := []corev1.VolumeMount{
			{Name: "raddb-certs", MountPath: "/etc/raddb/certs/ca.pem", SubPath: "ca.pem"},
			{Name: "raddb-certs", MountPath: "/etc/raddb/certs/server.key", SubPath: "server.key"},
			{Name: "raddb-certs", MountPath: "/etc/raddb/certs/server.pem", SubPath: "server.pem"},
		}
		if mountDH {
			mounts = append(mounts, corev1.VolumeMount{Name: "raddb-certs", MountPath: "/etc/raddb/certs/dh", SubPath: "dh"})
		}
		return mounts
	}
}

func layoutHasCA(layout certs.Layout) (struct{}, bool) {
	if layout.TLSHasCA {
		return struct{}{}, true
	}
	return struct{}{}, false
}
