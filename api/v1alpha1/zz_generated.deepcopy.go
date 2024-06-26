//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentsStatus) DeepCopyInto(out *ComponentsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentsStatus.
func (in *ComponentsStatus) DeepCopy() *ComponentsStatus {
	if in == nil {
		return nil
	}
	out := new(ComponentsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InvokerDeployment) DeepCopyInto(out *InvokerDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InvokerDeployment.
func (in *InvokerDeployment) DeepCopy() *InvokerDeployment {
	if in == nil {
		return nil
	}
	out := new(InvokerDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InvokerDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InvokerDeploymentList) DeepCopyInto(out *InvokerDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InvokerDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InvokerDeploymentList.
func (in *InvokerDeploymentList) DeepCopy() *InvokerDeploymentList {
	if in == nil {
		return nil
	}
	out := new(InvokerDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InvokerDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InvokerDeploymentSpec) DeepCopyInto(out *InvokerDeploymentSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.StatefulEntities != nil {
		in, out := &in.StatefulEntities, &out.StatefulEntities
		*out = make([]StatefulEntity, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.HistoryLimit != nil {
		in, out := &in.HistoryLimit, &out.HistoryLimit
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InvokerDeploymentSpec.
func (in *InvokerDeploymentSpec) DeepCopy() *InvokerDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(InvokerDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InvokerDeploymentStatus) DeepCopyInto(out *InvokerDeploymentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Components = in.Components
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InvokerDeploymentStatus.
func (in *InvokerDeploymentStatus) DeepCopy() *InvokerDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(InvokerDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodConfig) DeepCopyInto(out *PodConfig) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Partitions != nil {
		in, out := &in.Partitions, &out.Partitions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodConfig.
func (in *PodConfig) DeepCopy() *PodConfig {
	if in == nil {
		return nil
	}
	out := new(PodConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulEntity) DeepCopyInto(out *StatefulEntity) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.ExecutorImage != nil {
		in, out := &in.ExecutorImage, &out.ExecutorImage
		*out = new(string)
		**out = **in
	}
	if in.IOImage != nil {
		in, out := &in.IOImage, &out.IOImage
		*out = new(string)
		**out = **in
	}
	if in.StateImage != nil {
		in, out := &in.StateImage, &out.StateImage
		*out = new(string)
		**out = **in
	}
	if in.IOAddress != nil {
		in, out := &in.IOAddress, &out.IOAddress
		*out = new(string)
		**out = **in
	}
	if in.InputTopic != nil {
		in, out := &in.InputTopic, &out.InputTopic
		*out = new(string)
		**out = **in
	}
	if in.OutputTopic != nil {
		in, out := &in.OutputTopic, &out.OutputTopic
		*out = new(string)
		**out = **in
	}
	if in.Pods != nil {
		in, out := &in.Pods, &out.Pods
		*out = make([]PodConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulEntity.
func (in *StatefulEntity) DeepCopy() *StatefulEntity {
	if in == nil {
		return nil
	}
	out := new(StatefulEntity)
	in.DeepCopyInto(out)
	return out
}
