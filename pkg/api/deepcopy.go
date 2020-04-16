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

package api

// DeepCopy clones a CreateServiceInstanceRequest.
func (in *CreateServiceInstanceRequest) DeepCopy() *CreateServiceInstanceRequest {
	out := *in

	if in.Context != nil {
		out.Context = in.Context.DeepCopy()
	}

	if in.Parameters != nil {
		out.Parameters = in.Parameters.DeepCopy()
	}

	if in.MaintenanceInfo != nil {
		out.MaintenanceInfo = in.MaintenanceInfo.DeepCopy()
	}

	return &out
}

// DeepCopy clones a MaintenanceInfo.
func (in *MaintenanceInfo) DeepCopy() *MaintenanceInfo {
	out := *in

	return &out
}

// DeepCopy clones a UpdateServiceInstanceRequestPreviousValues.
func (in *UpdateServiceInstanceRequestPreviousValues) DeepCopy() *UpdateServiceInstanceRequestPreviousValues {
	out := *in

	if in.MaintenanceInfo != nil {
		out.MaintenanceInfo = in.MaintenanceInfo.DeepCopy()
	}

	return &out
}

// DeepCopy clones a UpdateServiceInstanceRequest.
func (in *UpdateServiceInstanceRequest) DeepCopy() *UpdateServiceInstanceRequest {
	out := *in

	if in.Context != nil {
		out.Context = in.Context.DeepCopy()
	}

	if in.Parameters != nil {
		out.Parameters = in.Parameters.DeepCopy()
	}

	if in.PreviousValues != nil {
		out.PreviousValues = in.PreviousValues.DeepCopy()
	}

	if in.MaintenanceInfo != nil {
		out.MaintenanceInfo = in.MaintenanceInfo.DeepCopy()
	}

	return &out
}

// DeepCopy clones a CreateServiceBindingRequest.
func (in *CreateServiceBindingRequest) DeepCopy() *CreateServiceBindingRequest {
	out := *in

	if in.Context != nil {
		out.Context = in.Context.DeepCopy()
	}

	if in.BindResource != nil {
		out.BindResource = in.BindResource.DeepCopy()
	}

	if in.Parameters != nil {
		out.Parameters = in.Parameters.DeepCopy()
	}

	return &out
}
