package api

import (
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// Error is the structured JSON response to send to a client on an error condition.
type Error struct {
	Error          string `json:"error,omitempty"`
	Description    string `json:"description,omitempty"`
	InstanceUsable *bool  `json:"instance_usable,omitempty"`
}

// CreateServiceInstanceRequest is submitted by the client when creating a service instance.
type CreateServiceInstanceRequest struct {
	ServiceID        string                `json:"service_id"`
	PlanID           string                `json:"plan_id"`
	Context          *runtime.RawExtension `json:"context"`
	OrganizationGUID string                `json:"organization_guid"`
	SpaceGUID        string                `json:"space_guid"`
	Parameters       *runtime.RawExtension `json:"parameters"`
	MaintenanceInfo  *v1.MaintenanceInfo   `json:"maintenance_info"`
}

// CreateServiceInstanceResponse is returned by the server when creating a service instance.
type CreateServiceInstanceResponse struct {
	DashboardURL string `json:"dashboard_url,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

// PollServiceInstanceResponse is returned by the server when an operation is being polled.
type PollServiceInstanceResponse struct {
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
}

// GetServiceInstanceResponse is returned by the server when a service instance is read.
type GetServiceInstanceResponse struct {
	ServiceID    string                `json:"service_id,omitempty"`
	PlanID       string                `json:"plan_id,omitempty"`
	DashboardURL string                `json:"dashboard_url,omitempty"`
	Parameters   *runtime.RawExtension `json:"parameters,omitempty"`
}

// UpdateServiceInstanceRequest is submitted by the client when updating a service instance.
type UpdateServiceInstanceRequest struct {
	Context         *runtime.RawExtension                       `json:"context,omitempty"`
	ServiceID       string                                      `json:"service_id"`
	PlanID          string                                      `json:"plan_id,omitempty"`
	Parameters      *runtime.RawExtension                       `json:"parameters,omitempty"`
	PreviousValues  *UpdateServiceInstanceRequestPreviousValues `json:"previous_values,omitempty"`
	MaintenanceInfo *v1.MaintenanceInfo                         `json:"maintenance_info,omitempty"`
}

// UpdateServiceInstanceRequestPreviousValues is additional information about the instance
// prior to an update.
type UpdateServiceInstanceRequestPreviousValues struct {
	ServiceID       string              `json:"service_id,omitempty"`
	PlanID          string              `json:"plan_id,omitempty"`
	OrganizationID  string              `json:"organization_id,omitempty"`
	SpaceID         string              `json:"space_id,omitempty"`
	MaintenanceInfo *v1.MaintenanceInfo `json:"maintenance_info,omitempty"`
}

// UpdateServiceInstanceResponse is returned by the server when updating a service instance.
type UpdateServiceInstanceResponse struct {
	DashboardURL string `json:"dashboard_url,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

// CreateServiceBindingRequest is provided by the client when it wishes to bind to the service
// instance and get credentials.
type CreateServiceBindingRequest struct {
	Context      *runtime.RawExtension `json:"context"`
	ServiceID    string                `json:"service_id"`
	PlanID       string                `json:"plan_id"`
	AppGUID      string                `json:"app_guid"`
	BindResource *runtime.RawExtension `json:"bind_resource"`
	Parameters   *runtime.RawExtension `json:"parameters"`
}

// CreateServiceBindingResponse is returned to the client when an aysnc request
// to create a binding is made.
type CreateServiceBindingResponse struct {
	Operation string `json:"operation"`
}

// PollServiceBindingResponse is returned by the server when an operation is being polled.
type PollServiceBindingResponse struct {
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
}
