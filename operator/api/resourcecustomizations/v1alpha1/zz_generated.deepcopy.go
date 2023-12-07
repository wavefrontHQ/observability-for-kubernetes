//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022.

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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceCustomization) DeepCopyInto(out *ResourceCustomization) {
	*out = *in
	in.Tolerations.DeepCopyInto(&out.Tolerations)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceCustomization.
func (in *ResourceCustomization) DeepCopy() *ResourceCustomization {
	if in == nil {
		return nil
	}
	out := new(ResourceCustomization)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceCustomizations) DeepCopyInto(out *ResourceCustomizations) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceCustomizations.
func (in *ResourceCustomizations) DeepCopy() *ResourceCustomizations {
	if in == nil {
		return nil
	}
	out := new(ResourceCustomizations)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceCustomizations) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceCustomizationsList) DeepCopyInto(out *ResourceCustomizationsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ResourceCustomizations, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceCustomizationsList.
func (in *ResourceCustomizationsList) DeepCopy() *ResourceCustomizationsList {
	if in == nil {
		return nil
	}
	out := new(ResourceCustomizationsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceCustomizationsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceCustomizationsSpec) DeepCopyInto(out *ResourceCustomizationsSpec) {
	*out = *in
	if in.All != nil {
		in, out := &in.All, &out.All
		*out = new(ResourceCustomization)
		(*in).DeepCopyInto(*out)
	}
	if in.ByName != nil {
		in, out := &in.ByName, &out.ByName
		*out = make(map[string]WorkloadCustomization, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceCustomizationsSpec.
func (in *ResourceCustomizationsSpec) DeepCopy() *ResourceCustomizationsSpec {
	if in == nil {
		return nil
	}
	out := new(ResourceCustomizationsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceCustomizationsStatus) DeepCopyInto(out *ResourceCustomizationsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceCustomizationsStatus.
func (in *ResourceCustomizationsStatus) DeepCopy() *ResourceCustomizationsStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceCustomizationsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tolerations) DeepCopyInto(out *Tolerations) {
	*out = *in
	if in.Add != nil {
		in, out := &in.Add, &out.Add
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Remove != nil {
		in, out := &in.Remove, &out.Remove
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tolerations.
func (in *Tolerations) DeepCopy() *Tolerations {
	if in == nil {
		return nil
	}
	out := new(Tolerations)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadCustomization) DeepCopyInto(out *WorkloadCustomization) {
	*out = *in
	in.ResourceCustomization.DeepCopyInto(&out.ResourceCustomization)
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadCustomization.
func (in *WorkloadCustomization) DeepCopy() *WorkloadCustomization {
	if in == nil {
		return nil
	}
	out := new(WorkloadCustomization)
	in.DeepCopyInto(out)
	return out
}
