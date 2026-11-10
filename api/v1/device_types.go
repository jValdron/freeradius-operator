package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeviceSpec defines the desired state of Device.
type DeviceSpec struct {
	// +kubebuilder:validation:MinItems=1
	MACAddresses []string `json:"macAddresses"`

	VLAN int32 `json:"vlan"`
}

// DeviceStatus defines the observed state of Device.
type DeviceStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=dev,scope=Namespaced

// Device is the Schema for the devices API.
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec"`
	Status DeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeviceList contains a list of Device.
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Device{}, &DeviceList{})
		return nil
	})
}
