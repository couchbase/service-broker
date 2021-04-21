// Copyright 2020-2021 Couchbase, Inc.
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

package v1alpha1

import (
	"github.com/couchbase/service-broker/pkg/api"
)

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in ServiceCatalog) Convert() api.ServiceCatalog {
	out := api.ServiceCatalog{}

	out.Services = make([]api.ServiceOffering, len(in.Services))
	for i, o := range in.Services {
		out.Services[i] = o.Convert()
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in ServiceOffering) Convert() api.ServiceOffering {
	out := api.ServiceOffering{
		Name:          in.Name,
		ID:            in.ID,
		Description:   in.Description,
		Tags:          in.Tags,
		Requires:      in.Requires,
		Bindable:      in.Bindable,
		Metadata:      in.Metadata,
		PlanUpdatable: in.PlanUpdatable,
	}

	if in.DashboardClient != nil {
		dashboardClient := in.DashboardClient.Convert()
		out.DashboardClient = &dashboardClient
	}

	out.Plans = make([]api.ServicePlan, len(in.Plans))
	for i, o := range in.Plans {
		out.Plans[i] = o.Convert()
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in DashboardClient) Convert() api.DashboardClient {
	return api.DashboardClient{
		ID:            in.ID,
		Secret:        in.Secret,
		RedirectedURI: in.RedirectedURI,
	}
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in ServicePlan) Convert() api.ServicePlan {
	out := api.ServicePlan{
		ID:          in.ID,
		Name:        in.Name,
		Description: in.Description,
		Metadata:    in.Metadata,
		Free:        in.Free,
		Bindable:    in.Bindable,
	}

	if in.Schemas != nil {
		schemas := in.Schemas.Convert()
		out.Schemas = &schemas
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in Schemas) Convert() api.Schemas {
	out := api.Schemas{}

	if in.ServiceInstance != nil {
		serviceInstance := in.ServiceInstance.Convert()
		out.ServiceInstance = &serviceInstance
	}

	if in.ServiceBinding != nil {
		serviceBinding := in.ServiceBinding.Convert()
		out.ServiceBinding = &serviceBinding
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in ServiceInstanceSchema) Convert() api.ServiceInstanceSchema {
	out := api.ServiceInstanceSchema{}

	if in.Create != nil {
		create := in.Create.Convert()
		out.Create = &create
	}

	if in.Update != nil {
		update := in.Update.Convert()
		out.Update = &update
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in ServiceBindingSchema) Convert() api.ServiceBindingSchema {
	out := api.ServiceBindingSchema{}

	if in.Create != nil {
		create := in.Create.Convert()
		out.Create = &create
	}

	return out
}

// Convert reformats a Kubernetes catalog object as an Open Service Broker object.
func (in InputParamtersSchema) Convert() api.InputParamtersSchema {
	return api.InputParamtersSchema{Parameters: in.Parameters}
}
