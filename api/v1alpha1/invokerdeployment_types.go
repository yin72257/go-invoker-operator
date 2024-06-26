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
	Name       *string  `json:"name,omitempty"`
	Partitions []string `json:"partitions,omitempty"`
	// Configuration used for InvokerDeployment
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

type StatefulEntity struct {
	Name          *string `json:"name,omitempty"`
	ExecutorImage *string `json:"executorImage,omitempty"`
	IOImage       *string `json:"ioImage,omitempty"`
	StateImage    *string `json:"stateImage,omitempty"`

	IOAddress   *string `json:"ioAddress,omitempty"`
	InputTopic  *string `json:"inputTopic,omitempty"`
	OutputTopic *string `json:"outputTopic,omitempty"`
	// Default: 1
	// +optional
	// +kubebuilder:validation:MinItems:=1
	Pods []PodConfig `json:"pods,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InvokerDeploymentSpec defines the desired state of InvokerDeployment
type InvokerDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of InvokerDeployment being deployed
	Version string `json:"version,omitempty"`

	// Image pull policy: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// +kubebuilder:validation:Enum"";Always;Never;IfNotPresent
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// An optional list of references to Secrets
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	StatefulEntities []StatefulEntity `json:"statefulEntities,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	HistoryLimit *int32 `json:"historyLimit,omitempty"`
}

type ComponentsStatus struct {
	ConfigMap string `json:"configMap"`

	StatefulEntities string `json:"statefulEntities"`

	Secret string `json:"secret"`
}

// InvokerDeploymentStatus defines the observed state of InvokerDeployment
type InvokerDeploymentStatus struct {
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

// InvokerDeployment is the Schema for the InvokerDeployments API
type InvokerDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InvokerDeploymentSpec   `json:"spec,omitempty"`
	Status InvokerDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InvokerDeploymentList contains a list of InvokerDeployment
type InvokerDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InvokerDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InvokerDeployment{}, &InvokerDeploymentList{})
}
