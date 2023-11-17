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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Auth) DeepCopyInto(out *Auth) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Auth.
func (in *Auth) DeepCopy() *Auth {
	if in == nil {
		return nil
	}
	out := new(Auth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoTracingPixie) DeepCopyInto(out *AutoTracingPixie) {
	*out = *in
	out.PixieShared = in.PixieShared
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoTracingPixie.
func (in *AutoTracingPixie) DeepCopy() *AutoTracingPixie {
	if in == nil {
		return nil
	}
	out := new(AutoTracingPixie)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Collector) DeepCopyInto(out *Collector) {
	*out = *in
	out.Resources = in.Resources
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Collector.
func (in *Collector) DeepCopy() *Collector {
	if in == nil {
		return nil
	}
	out := new(Collector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControlPlane) DeepCopyInto(out *ControlPlane) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControlPlane.
func (in *ControlPlane) DeepCopy() *ControlPlane {
	if in == nil {
		return nil
	}
	out := new(ControlPlane)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetStatus) DeepCopyInto(out *DaemonSetStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetStatus.
func (in *DaemonSetStatus) DeepCopy() *DaemonSetStatus {
	if in == nil {
		return nil
	}
	out := new(DaemonSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataCollection) DeepCopyInto(out *DataCollection) {
	*out = *in
	in.Metrics.DeepCopyInto(&out.Metrics)
	in.Logging.DeepCopyInto(&out.Logging)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]Toleration, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataCollection.
func (in *DataCollection) DeepCopy() *DataCollection {
	if in == nil {
		return nil
	}
	out := new(DataCollection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataExport) DeepCopyInto(out *DataExport) {
	*out = *in
	out.ExternalWavefrontProxy = in.ExternalWavefrontProxy
	out.WavefrontProxy = in.WavefrontProxy
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataExport.
func (in *DataExport) DeepCopy() *DataExport {
	if in == nil {
		return nil
	}
	out := new(DataExport)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentStatus) DeepCopyInto(out *DeploymentStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentStatus.
func (in *DeploymentStatus) DeepCopy() *DeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(DeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Experimental) DeepCopyInto(out *Experimental) {
	*out = *in
	out.Pixie = in.Pixie
	out.Autotracing = in.Autotracing
	out.Insights = in.Insights
	out.Hub = in.Hub
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Experimental.
func (in *Experimental) DeepCopy() *Experimental {
	if in == nil {
		return nil
	}
	out := new(Experimental)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalWavefrontProxy) DeepCopyInto(out *ExternalWavefrontProxy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalWavefrontProxy.
func (in *ExternalWavefrontProxy) DeepCopy() *ExternalWavefrontProxy {
	if in == nil {
		return nil
	}
	out := new(ExternalWavefrontProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Filters) DeepCopyInto(out *Filters) {
	*out = *in
	if in.DenyList != nil {
		in, out := &in.DenyList, &out.DenyList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AllowList != nil {
		in, out := &in.AllowList, &out.AllowList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TagGuaranteeList != nil {
		in, out := &in.TagGuaranteeList, &out.TagGuaranteeList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TagDenyList != nil {
		in, out := &in.TagDenyList, &out.TagDenyList
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.TagAllowList != nil {
		in, out := &in.TagAllowList, &out.TagAllowList
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.TagExclude != nil {
		in, out := &in.TagExclude, &out.TagExclude
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TagInclude != nil {
		in, out := &in.TagInclude, &out.TagInclude
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Filters.
func (in *Filters) DeepCopy() *Filters {
	if in == nil {
		return nil
	}
	out := new(Filters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Histogram) DeepCopyInto(out *Histogram) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Histogram.
func (in *Histogram) DeepCopy() *Histogram {
	if in == nil {
		return nil
	}
	out := new(Histogram)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpProxy) DeepCopyInto(out *HttpProxy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpProxy.
func (in *HttpProxy) DeepCopy() *HttpProxy {
	if in == nil {
		return nil
	}
	out := new(HttpProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Hub) DeepCopyInto(out *Hub) {
	*out = *in
	out.Pixie = in.Pixie
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Hub.
func (in *Hub) DeepCopy() *Hub {
	if in == nil {
		return nil
	}
	out := new(Hub)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Insights) DeepCopyInto(out *Insights) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Insights.
func (in *Insights) DeepCopy() *Insights {
	if in == nil {
		return nil
	}
	out := new(Insights)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerTracing) DeepCopyInto(out *JaegerTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerTracing.
func (in *JaegerTracing) DeepCopy() *JaegerTracing {
	if in == nil {
		return nil
	}
	out := new(JaegerTracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogFilters) DeepCopyInto(out *LogFilters) {
	*out = *in
	if in.TagDenyList != nil {
		in, out := &in.TagDenyList, &out.TagDenyList
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.TagAllowList != nil {
		in, out := &in.TagAllowList, &out.TagAllowList
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogFilters.
func (in *LogFilters) DeepCopy() *LogFilters {
	if in == nil {
		return nil
	}
	out := new(LogFilters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Logging) DeepCopyInto(out *Logging) {
	*out = *in
	in.Filters.DeepCopyInto(&out.Filters)
	out.Resources = in.Resources
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Logging.
func (in *Logging) DeepCopy() *Logging {
	if in == nil {
		return nil
	}
	out := new(Logging)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metrics) DeepCopyInto(out *Metrics) {
	*out = *in
	out.ControlPlane = in.ControlPlane
	in.Filters.DeepCopyInto(&out.Filters)
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.ClusterCollector = in.ClusterCollector
	out.NodeCollector = in.NodeCollector
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metrics.
func (in *Metrics) DeepCopy() *Metrics {
	if in == nil {
		return nil
	}
	out := new(Metrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OTLP) DeepCopyInto(out *OTLP) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OTLP.
func (in *OTLP) DeepCopy() *OTLP {
	if in == nil {
		return nil
	}
	out := new(OTLP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Pixie) DeepCopyInto(out *Pixie) {
	*out = *in
	out.TableStoreLimits = in.TableStoreLimits
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Pixie.
func (in *Pixie) DeepCopy() *Pixie {
	if in == nil {
		return nil
	}
	out := new(Pixie)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PixieShared) DeepCopyInto(out *PixieShared) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PixieShared.
func (in *PixieShared) DeepCopy() *PixieShared {
	if in == nil {
		return nil
	}
	out := new(PixieShared)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreprocessorRules) DeepCopyInto(out *PreprocessorRules) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreprocessorRules.
func (in *PreprocessorRules) DeepCopy() *PreprocessorRules {
	if in == nil {
		return nil
	}
	out := new(PreprocessorRules)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resource) DeepCopyInto(out *Resource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resource.
func (in *Resource) DeepCopy() *Resource {
	if in == nil {
		return nil
	}
	out := new(Resource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceStatus) DeepCopyInto(out *ResourceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceStatus.
func (in *ResourceStatus) DeepCopy() *ResourceStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resources) DeepCopyInto(out *Resources) {
	*out = *in
	out.Requests = in.Requests
	out.Limits = in.Limits
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resources.
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TableStoreLimits) DeepCopyInto(out *TableStoreLimits) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TableStoreLimits.
func (in *TableStoreLimits) DeepCopy() *TableStoreLimits {
	if in == nil {
		return nil
	}
	out := new(TableStoreLimits)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Toleration) DeepCopyInto(out *Toleration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Toleration.
func (in *Toleration) DeepCopy() *Toleration {
	if in == nil {
		return nil
	}
	out := new(Toleration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tracing) DeepCopyInto(out *Tracing) {
	*out = *in
	out.Wavefront = in.Wavefront
	out.Jaeger = in.Jaeger
	out.Zipkin = in.Zipkin
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tracing.
func (in *Tracing) DeepCopy() *Tracing {
	if in == nil {
		return nil
	}
	out := new(Tracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Wavefront) DeepCopyInto(out *Wavefront) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Wavefront.
func (in *Wavefront) DeepCopy() *Wavefront {
	if in == nil {
		return nil
	}
	out := new(Wavefront)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Wavefront) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontList) DeepCopyInto(out *WavefrontList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Wavefront, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontList.
func (in *WavefrontList) DeepCopy() *WavefrontList {
	if in == nil {
		return nil
	}
	out := new(WavefrontList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WavefrontList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontProxy) DeepCopyInto(out *WavefrontProxy) {
	*out = *in
	out.Tracing = in.Tracing
	out.OTLP = in.OTLP
	out.Histogram = in.Histogram
	out.Resources = in.Resources
	out.HttpProxy = in.HttpProxy
	out.PreprocessorRules = in.PreprocessorRules
	out.Auth = in.Auth
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontProxy.
func (in *WavefrontProxy) DeepCopy() *WavefrontProxy {
	if in == nil {
		return nil
	}
	out := new(WavefrontProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontSpec) DeepCopyInto(out *WavefrontSpec) {
	*out = *in
	out.DataExport = in.DataExport
	in.DataCollection.DeepCopyInto(&out.DataCollection)
	out.Experimental = in.Experimental
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontSpec.
func (in *WavefrontSpec) DeepCopy() *WavefrontSpec {
	if in == nil {
		return nil
	}
	out := new(WavefrontSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontStatus) DeepCopyInto(out *WavefrontStatus) {
	*out = *in
	if in.ResourceStatuses != nil {
		in, out := &in.ResourceStatuses, &out.ResourceStatuses
		*out = make([]ResourceStatus, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontStatus.
func (in *WavefrontStatus) DeepCopy() *WavefrontStatus {
	if in == nil {
		return nil
	}
	out := new(WavefrontStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontTracing) DeepCopyInto(out *WavefrontTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontTracing.
func (in *WavefrontTracing) DeepCopy() *WavefrontTracing {
	if in == nil {
		return nil
	}
	out := new(WavefrontTracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ZipkinTracing) DeepCopyInto(out *ZipkinTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ZipkinTracing.
func (in *ZipkinTracing) DeepCopy() *ZipkinTracing {
	if in == nil {
		return nil
	}
	out := new(ZipkinTracing)
	in.DeepCopyInto(out)
	return out
}