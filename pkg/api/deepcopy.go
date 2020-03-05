package api

// DeepCopy clones a CreateServiceInstanceRequest
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
