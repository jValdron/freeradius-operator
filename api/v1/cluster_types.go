package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {
	// When true, unlabeled Client/User/Device resources are assigned to this cluster.
	// +optional
	IsDefaultCluster *bool `json:"isDefaultCluster,omitempty"`

	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// FreeRADIUS container image.
	// +kubebuilder:default="freeradius/freeradius-server:3.2.10-alpine"
	// +optional
	Image string `json:"image,omitempty"`

	// Run radiusd with -X (verbose logging).
	// +optional
	Debug bool `json:"debug,omitempty"`

	Service ServiceSpec `json:"service"`

	// VLAN for authorize DEFAULT when set.
	// +optional
	DefaultVlan *int32 `json:"defaultVlan,omitempty"`

	Certificate CertificateSpec `json:"certificate"`

	// +optional
	EAP *EAPSpec `json:"eap,omitempty"`

	// +optional
	LDAP *LDAPSpec `json:"ldap,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// Cluster is the Schema for the clusters API.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Cluster{}, &ClusterList{})
		return nil
	})
}
