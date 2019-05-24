package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ToolChainEnablerSpec defines the desired state of ToolChainEnabler
type ToolChainEnablerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	AuthURL             string `json:"authURL"`
	ClusterURL          string `json:"clusterURL"`
	ClusterName         string `json:"clusterName"`
	ToolChainSecretName string `json:"toolChainSecretName"`
}

// ToolChainEnablerStatus defines the observed state of ToolChainEnabler
type ToolChainEnablerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ToolChainEnabler is the Schema for the toolchainenablers API
// +k8s:openapi-gen=true
type ToolChainEnabler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolChainEnablerSpec   `json:"spec,omitempty"`
	Status ToolChainEnablerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ToolChainEnablerList contains a list of ToolChainEnabler
type ToolChainEnablerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolChainEnabler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolChainEnabler{}, &ToolChainEnablerList{})
}
