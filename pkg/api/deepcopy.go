package api

// DeepCopy clones a CreateServiceInstanceRequest
func (i *CreateServiceInstanceRequest) DeepCopy() *CreateServiceInstanceRequest {
	return &CreateServiceInstanceRequest{
		ServiceID:        i.ServiceID,
		PlanID:           i.PlanID,
		Context:          i.Context.DeepCopy(),
		OrganizationGUID: i.OrganizationGUID,
		SpaceGUID:        i.SpaceGUID,
		Parameters:       i.Parameters.DeepCopy(),
		MaintenanceInfo:  i.MaintenanceInfo.DeepCopy(),
	}
}
