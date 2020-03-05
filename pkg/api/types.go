package api

import (
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// ErrorType is returned when a service broker error is encountered.
type ErrorType string

const (
	// ErrorAsyncRequired meand this request requires client support for asynchronous
	// service operations.
	ErrorAsyncRequired ErrorType = "AsyncRequired"

	// ErrorConcurrencyError means the Service Broker does not support concurrent
	// requests that mutate the same resource.
	ErrorConcurrencyError ErrorType = "ConcurrencyError"

	// ErrorRequiresApp means the request body is missing the app_guid field.
	ErrorRequiresApp ErrorType = "RequiresApp"

	// ErrorMaintenanceInfoConflict means the maintenance_info.version field provided
	// in the request does not match the maintenance_info.version field provided in
	// the Service Broker's Catalog.
	ErrorMaintenanceInfoConflict ErrorType = "MaintenanceInfoConflict"

	// ErrorInternalServerError means that something that shouldn't ever break has.
	ErrorInternalServerError ErrorType = "InternalServerError"

	// ErrorConfigurationError means that the broker has been misconfigured.
	ErrorConfigurationError ErrorType = "ConfigurationError"

	// ErrorQueryError means that the user specified query is inavlid.
	ErrorQueryError ErrorType = "QueryError"

	// ErrorParameterError means that the user specified parameters are
	// invalid.
	ErrorParameterError ErrorType = "ParameterError"

	// ErrorValidationError means that the supplied parameters failed JSON schema
	// validation.
	ErrorValidationError ErrorType = "ValidationError"

	// ErrorResourceConflict means that an attempt to create a resource has resulted
	// in a conflict with an existing one.
	ErrorResourceConflict ErrorType = "ResourceConflict"

	// ErrorResourceNotFound means that an attempt has been made to access a resource
	// that does not extst.
	ErrorResourceNotFound ErrorType = "ResourceNotFound"

	// ErrorResourceGone means that a delete request has failed because the
	// requested resource does not exist.
	ErrorResourceGone ErrorType = "ResourceGone"
)

// PollState is returned when an asynchronous request is polled.
type PollState string

const (
	// PollStateInProgress means the async request is still being done.
	PollStateInProgress PollState = "in progress"

	// PollStateInProgress means the async request completed successfully.
	PollStateSucceeded PollState = "succeeded"

	// PollStateFailed means the async request failed.
	PollStateFailed PollState = "failed"
)

// Error is the structured JSON response to send to a client on an error condition.
type Error struct {
	// A single word in camel case that uniquely identifies the error condition.
	// If present, MUST be a non-empty string.
	Error ErrorType `json:"error,omitempty"`

	// A user-facing error message explaining why the request failed.
	// If present, MUST be a non-empty string.
	Description string `json:"description,omitempty"`

	// If an update or deprovisioning operation failed, this flag indicates
	// whether or not the Service Instance is still usable. If true, the
	// Service Instance can still be used, false otherwise. This field MUST NOT
	// be present for errors of other operations. Defaults to true.
	InstanceUsable *bool `json:"instance_usable,omitempty"`
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
	State       PollState `json:"state"`
	Description string    `json:"description,omitempty"`
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
	State       PollState `json:"state"`
	Description string    `json:"description,omitempty"`
}
