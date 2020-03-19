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
