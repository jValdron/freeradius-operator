package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClientSpec defines the desired state of Client.
type ClientSpec struct {
	// +kubebuilder:validation:MinLength=1
	IPAddress string `json:"ipAddress"`

	// +optional
	Secret string `json:"secret,omitempty"`

	// Secret data key: secret.
	// +optional
	FromSecretRef *SecretRef `json:"fromSecretRef,omitempty"`
}

// ClientStatus defines the observed state of Client.
type ClientStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// Client is the Schema for the clients API.
type Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientSpec   `json:"spec"`
	Status ClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClientList contains a list of Client.
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Client{}, &ClientList{})
		return nil
	})
}
