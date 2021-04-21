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

package api

// ServiceCatalog is returned from /v2/catalog.
type ServiceCatalog struct {
	Services []ServiceOffering `json:"services"`
}

// ServiceOffering must be provided by a service catalog.
type ServiceOffering struct {
	Name            string           `json:"name"`
	ID              string           `json:"id"`
	Description     string           `json:"description"`
	Tags            []string         `json:"tags,omitempty"`
	Requires        []string         `json:"requires,omitempty"`
	Bindable        bool             `json:"bindable"`
	Metadata        interface{}      `json:"metadata,omitempty"`
	DashboardClient *DashboardClient `json:"dashboard_client,omitempty"`
	PlanUpdatable   bool             `json:"plan_updatable,omitempty"`
	Plans           []ServicePlan    `json:"plans"`
}

// DashboardClient may be provided by a service offering.
type DashboardClient struct {
	ID            string `json:"id"`
	Secret        string `json:"secret"`
	RedirectedURI string `json:"redirected_uri,omitempty"`
}

// ServicePlan must be provided by a service offering.
type ServicePlan struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Metadata    interface{} `json:"metadata,omitempty"`
	Free        bool        `json:"free,omitempty"`
	Bindable    *bool       `json:"bindable,omitempty"`
	Schemas     *Schemas    `json:"schemas,omitempty"`
}

// Schemas may be provided for a service plan.
type Schemas struct {
	ServiceInstance *ServiceInstanceSchema `json:"service_instance,omitempty"`
	ServiceBinding  *ServiceBindingSchema  `json:"service_binding,omitempty"`
}

// ServiceInstanceSchema may be provided for a service plan.
type ServiceInstanceSchema struct {
	Create *InputParamtersSchema `json:"create,omitempty"`
	Update *InputParamtersSchema `json:"update,omitempty"`
}

// ServiceBindingSchema may be provided for a service plan.
type ServiceBindingSchema struct {
	Create *InputParamtersSchema `json:"create,omitempty"`
}

// InputParamtersSchema may be provided for a service plan.
type InputParamtersSchema struct {
	Parameters interface{} `json:"parameters,omitempty"`
}
