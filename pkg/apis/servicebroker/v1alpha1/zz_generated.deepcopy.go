// +build !ignore_autogenerated

// Copyright 2020 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigurationBinding) DeepCopyInto(out *ConfigurationBinding) {
	*out = *in
	in.ServiceInstance.DeepCopyInto(&out.ServiceInstance)
	if in.ServiceBinding != nil {
		in, out := &in.ServiceBinding, &out.ServiceBinding
		*out = new(ServiceBrokerTemplateList)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigurationBinding.
func (in *ConfigurationBinding) DeepCopy() *ConfigurationBinding {
	if in == nil {
		return nil
	}
	out := new(ConfigurationBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigurationReadinessCheck) DeepCopyInto(out *ConfigurationReadinessCheck) {
	*out = *in
	if in.Condition != nil {
		in, out := &in.Condition, &out.Condition
		*out = new(ConfigurationReadinessCheckCondition)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigurationReadinessCheck.
func (in *ConfigurationReadinessCheck) DeepCopy() *ConfigurationReadinessCheck {
	if in == nil {
		return nil
	}
	out := new(ConfigurationReadinessCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigurationReadinessCheckCondition) DeepCopyInto(out *ConfigurationReadinessCheckCondition) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigurationReadinessCheckCondition.
func (in *ConfigurationReadinessCheckCondition) DeepCopy() *ConfigurationReadinessCheckCondition {
	if in == nil {
		return nil
	}
	out := new(ConfigurationReadinessCheckCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigurationTemplate) DeepCopyInto(out *ConfigurationTemplate) {
	*out = *in
	if in.Template != nil {
		in, out := &in.Template, &out.Template
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigurationTemplate.
func (in *ConfigurationTemplate) DeepCopy() *ConfigurationTemplate {
	if in == nil {
		return nil
	}
	out := new(ConfigurationTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DashboardClient) DeepCopyInto(out *DashboardClient) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DashboardClient.
func (in *DashboardClient) DeepCopy() *DashboardClient {
	if in == nil {
		return nil
	}
	out := new(DashboardClient)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InputParamtersSchema) DeepCopyInto(out *InputParamtersSchema) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InputParamtersSchema.
func (in *InputParamtersSchema) DeepCopy() *InputParamtersSchema {
	if in == nil {
		return nil
	}
	out := new(InputParamtersSchema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MaintenanceInfo) DeepCopyInto(out *MaintenanceInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MaintenanceInfo.
func (in *MaintenanceInfo) DeepCopy() *MaintenanceInfo {
	if in == nil {
		return nil
	}
	out := new(MaintenanceInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegistryValue) DeepCopyInto(out *RegistryValue) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegistryValue.
func (in *RegistryValue) DeepCopy() *RegistryValue {
	if in == nil {
		return nil
	}
	out := new(RegistryValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Schemas) DeepCopyInto(out *Schemas) {
	*out = *in
	if in.ServiceInstance != nil {
		in, out := &in.ServiceInstance, &out.ServiceInstance
		*out = new(ServiceInstanceSchema)
		(*in).DeepCopyInto(*out)
	}
	if in.ServiceBinding != nil {
		in, out := &in.ServiceBinding, &out.ServiceBinding
		*out = new(ServiceBindingSchema)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Schemas.
func (in *Schemas) DeepCopy() *Schemas {
	if in == nil {
		return nil
	}
	out := new(Schemas)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBindingSchema) DeepCopyInto(out *ServiceBindingSchema) {
	*out = *in
	if in.Create != nil {
		in, out := &in.Create, &out.Create
		*out = new(InputParamtersSchema)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBindingSchema.
func (in *ServiceBindingSchema) DeepCopy() *ServiceBindingSchema {
	if in == nil {
		return nil
	}
	out := new(ServiceBindingSchema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerConfig) DeepCopyInto(out *ServiceBrokerConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerConfig.
func (in *ServiceBrokerConfig) DeepCopy() *ServiceBrokerConfig {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceBrokerConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerConfigCondition) DeepCopyInto(out *ServiceBrokerConfigCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerConfigCondition.
func (in *ServiceBrokerConfigCondition) DeepCopy() *ServiceBrokerConfigCondition {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerConfigCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerConfigList) DeepCopyInto(out *ServiceBrokerConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ServiceBrokerConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerConfigList.
func (in *ServiceBrokerConfigList) DeepCopy() *ServiceBrokerConfigList {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceBrokerConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerConfigSpec) DeepCopyInto(out *ServiceBrokerConfigSpec) {
	*out = *in
	in.Catalog.DeepCopyInto(&out.Catalog)
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]ConfigurationTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Bindings != nil {
		in, out := &in.Bindings, &out.Bindings
		*out = make([]ConfigurationBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerConfigSpec.
func (in *ServiceBrokerConfigSpec) DeepCopy() *ServiceBrokerConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerConfigStatus) DeepCopyInto(out *ServiceBrokerConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ServiceBrokerConfigCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerConfigStatus.
func (in *ServiceBrokerConfigStatus) DeepCopy() *ServiceBrokerConfigStatus {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceBrokerTemplateList) DeepCopyInto(out *ServiceBrokerTemplateList) {
	*out = *in
	if in.Registry != nil {
		in, out := &in.Registry, &out.Registry
		*out = make([]RegistryValue, len(*in))
		copy(*out, *in)
	}
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ReadinessChecks != nil {
		in, out := &in.ReadinessChecks, &out.ReadinessChecks
		*out = make([]ConfigurationReadinessCheck, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceBrokerTemplateList.
func (in *ServiceBrokerTemplateList) DeepCopy() *ServiceBrokerTemplateList {
	if in == nil {
		return nil
	}
	out := new(ServiceBrokerTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceCatalog) DeepCopyInto(out *ServiceCatalog) {
	*out = *in
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]ServiceOffering, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceCatalog.
func (in *ServiceCatalog) DeepCopy() *ServiceCatalog {
	if in == nil {
		return nil
	}
	out := new(ServiceCatalog)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceInstanceSchema) DeepCopyInto(out *ServiceInstanceSchema) {
	*out = *in
	if in.Create != nil {
		in, out := &in.Create, &out.Create
		*out = new(InputParamtersSchema)
		(*in).DeepCopyInto(*out)
	}
	if in.Update != nil {
		in, out := &in.Update, &out.Update
		*out = new(InputParamtersSchema)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceInstanceSchema.
func (in *ServiceInstanceSchema) DeepCopy() *ServiceInstanceSchema {
	if in == nil {
		return nil
	}
	out := new(ServiceInstanceSchema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceOffering) DeepCopyInto(out *ServiceOffering) {
	*out = *in
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Requires != nil {
		in, out := &in.Requires, &out.Requires
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.DashboardClient != nil {
		in, out := &in.DashboardClient, &out.DashboardClient
		*out = new(DashboardClient)
		**out = **in
	}
	if in.Plans != nil {
		in, out := &in.Plans, &out.Plans
		*out = make([]ServicePlan, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceOffering.
func (in *ServiceOffering) DeepCopy() *ServiceOffering {
	if in == nil {
		return nil
	}
	out := new(ServiceOffering)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServicePlan) DeepCopyInto(out *ServicePlan) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.Bindable != nil {
		in, out := &in.Bindable, &out.Bindable
		*out = new(bool)
		**out = **in
	}
	if in.Schemas != nil {
		in, out := &in.Schemas, &out.Schemas
		*out = new(Schemas)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServicePlan.
func (in *ServicePlan) DeepCopy() *ServicePlan {
	if in == nil {
		return nil
	}
	out := new(ServicePlan)
	in.DeepCopyInto(out)
	return out
}
