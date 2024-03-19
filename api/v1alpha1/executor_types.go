/*
Copyright 2024.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ComponentStateNotReady = "NotReady"
	ComponentStateReady    = "Ready"
	ComponentStateUpdating = "Updating"
	ComponentStateDeleted  = "Deleted"
)

const (
	StateCreating    = "Creating"
	StateRunning     = "Running"
	StateUpdating    = "Updating"
	StateReconciling = "Reconciling"
	StateStopping    = "Stopping"
	StateStopped     = "Stopped"
)

type PodConfig struct {
	Partitions []string `json:"partitions,omitempty"`
}

type StatefulEntity struct {
	BrokerIp      *string `json:"brokerIp,omitempty"`
	Image         *string `json:"image,omitempty"`
	InputSidecar  *string `json:"inputSidecar,omitempty"`
	InputTopic    *string `json:"inputTopic,omitempty"`
	StateSidecar  *string `json:"stateSidecar,omitempty"`
	OutputSidecar *string `json:"outputSidecar,omitempty"`
	OutputTopic   *string `json:"outputTopic,omitempty"`
	Name          *string `json:"name,omitempty"`
	// Default: 1
	// +optional
	// +kubebuilder:validation:MinItems:=1
	Topology []PodConfig `json:"topology,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExecutorSpec defines the desired state of Executor
type ExecutorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of Executor being deployed
	Version string `json:"version,omitempty"`

	// Image pull policy: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// +kubebuilder:validation:Enum"";Always;Never;IfNotPresent
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// An optional list of references to Secrets
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Configuration used for executor
	// +optional
	Resource v1.ResourceRequirements `json:"resource,omitempty"`

	StatefulEntities []StatefulEntity `json:"statefulEntities,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	HistoryLimit *int32 `json:"historyLimit,omitempty"`
}

type ComponentsStatus struct {
	ConfigMap string `json:"configMap"`

	StatefulSet string `json:"statefulSet"`

	EntryService string `json:"entryService"`

	Secret string `json:"secret"`

	Ingress string `json:"ingress"`
}

// ExecutorStatus defines the observed state of Executor
type ExecutorStatus struct {
	// Conditions of the instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Components ComponentsStatus `json:"components"`

	State string `json:"state"`

	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	CurrentRevision string `json:"currentRevision"`

	NextRevision string `json:"nextRevision"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Executor is the Schema for the executors API
type Executor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutorSpec   `json:"spec,omitempty"`
	Status ExecutorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExecutorList contains a list of Executor
type ExecutorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Executor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Executor{}, &ExecutorList{})
}
