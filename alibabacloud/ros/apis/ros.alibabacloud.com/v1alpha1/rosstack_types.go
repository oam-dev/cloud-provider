package v1alpha1

import (
	oam "github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// RosStack is the Schema for the ROS API
type RosStack struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	//in our case, we use the same spec with applicationConfiguration
	Spec   oam.ApplicationConfigurationSpec   `json:"spec,omitempty"`
	Status oam.ApplicationConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type RosStackList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []RosStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RosStack{}, &RosStackList{})
}
