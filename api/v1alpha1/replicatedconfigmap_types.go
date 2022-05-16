/*
Copyright 2022 Rafael Brito.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// This is a replica of ConfigMap, so no Spec struct

type Phase string

const (
	PhaseNew        Phase = "New"
	PhaseInProgress Phase = "InProgress"
	PhaseFailed     Phase = "Failed"
	PhaseCompleted  Phase = "Completed"
	PhaseDeletion   Phase = "DeletionInProgress"
)

// ReplicatedConfigMapStatus defines the observed state of ReplicatedConfigMap
type ReplicatedConfigMapStatus struct {
	Phase              Phase  `json:"phase,omitempty"`
	MatchingNamespaces string `json:"matchingNamespaces,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ReplicatedConfigMap is the Schema for the replicatedconfigmaps API
type ReplicatedConfigMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go
	// With the exception of the immutability

	// Data contains the configuration data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// Values with non-UTF-8 byte sequences must use the BinaryData field.
	// The keys stored in Data must not overlap with the keys in
	// the BinaryData field, this is enforced during validation process.
	// +optional
	Data map[string]string `json:"data,omitempty"`

	// BinaryData contains the binary data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// BinaryData can contain byte sequences that are not in the UTF-8 range.
	// The keys stored in BinaryData must not overlap with the ones in
	// the Data field, this is enforced during validation process.
	// Using this field will require 1.10+ apiserver and
	// kubelet.
	// +optional
	BinaryData map[string][]byte `json:"binaryData,omitempty"`

	Status ReplicatedConfigMapStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReplicatedConfigMapList contains a list of ReplicatedConfigMap
type ReplicatedConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReplicatedConfigMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReplicatedConfigMap{}, &ReplicatedConfigMapList{})
}
