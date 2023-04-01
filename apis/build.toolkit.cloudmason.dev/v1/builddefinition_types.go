/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildDefinitionSpec defines the desired state of BuildDefinition
type BuildDefinitionSpec struct {
	// name of the gitRepository resource to reference
	GitRepository string `json:"gitRepository,omitempty"`
	// path to the dockerfile to build
	ContextPath string `json:"contextPath,omitempty"`
	// name of registry to push to (dockerhub, quay, etc)
	Registry string `json:"registry,omitempty"`
	// to get local secret needs to be updates.
	SecretRef *meta.LocalObjectReference `json:"secretRef,omitempty"`
	// this will need to have data about how to push to the registry
}

// BuildDefinitionStatus defines the observed state of BuildDefinition
type BuildDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BuildDefinition is the Schema for the builddefinitions API
type BuildDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildDefinitionSpec   `json:"spec,omitempty"`
	Status BuildDefinitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildDefinitionList contains a list of BuildDefinition
type BuildDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildDefinition{}, &BuildDefinitionList{})
}
