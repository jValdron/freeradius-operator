package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// UserSpec defines the desired state of User.
type UserSpec struct {
	// +optional
	Username string `json:"username,omitempty"`

	// +optional
	Password string `json:"password,omitempty"`

	// Secret data keys: username, password.
	// +optional
	FromSecretRef *SecretRef `json:"fromSecretRef,omitempty"`
}

// UserStatus defines the observed state of User.
type UserStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// User is the Schema for the users API.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &User{}, &UserList{})
		return nil
	})
}
