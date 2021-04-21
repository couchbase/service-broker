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

package broker

import (
	goerrors "errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/provisioners"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/apimachinery/pkg/runtime"
)

// ErrUnexpected is highly unlikely to happen...
var ErrUnexpected = goerrors.New("unexpected error")

// handleReadyz is a handler for Kubernetes readiness checks.  It is less verbose than the
// other API calls as it's called significantly more often.
func handleReadyz(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	httpResponse(w, http.StatusOK)
}

// handleReadCatalog advertises the classes of service we offer, and specifc plans to
// implement those classes.
func handleReadCatalog(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	JSONResponse(w, http.StatusOK, config.Config().Spec.Catalog.Convert())
}

// handleCreateServiceInstance creates a service instance of a plan.
func handleCreateServiceInstance(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Ensure the client supports async operation.
		if err := asyncRequired(r); err != nil {
			jsonError(w, err)
			return
		}

		// Parse the creation request.
		request := &api.CreateServiceInstanceRequest{}
		if err := jsonRequest(r, request); err != nil {
			jsonError(w, err)
			return
		}

		// Check parameters.
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		if err := validateServicePlan(config.Config(), request.ServiceID, request.PlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := validateParameters(config.Config(), request.ServiceID, request.PlanID, schemaTypeServiceInstance, schemaOperationCreate, request.Parameters); err != nil {
			jsonError(w, err)
			return
		}

		dirent, err := registerDirectoryInstance(config.Config(), request.Context, configuration.Namespace, instanceID, request.ServiceID, request.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		// Check if the instance already exists.
		entry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		if entry.Exists() {
			// If the instance already exists either return 200 if provisioned or
			// a 202 if it is still provisioning, or a 409 if provisioned or
			// provisioning with different attributes.
			serviceID, ok, err := entry.GetString(registry.ServiceID)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing service ID", ErrUnexpected))
				return
			}

			if serviceID != request.ServiceID {
				jsonError(w, errors.NewResourceConflictError("service ID %s does not match existing value %s", request.ServiceID, serviceID))
				return
			}

			planID, ok, err := entry.GetString(registry.PlanID)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
				return
			}

			if planID != request.PlanID {
				jsonError(w, errors.NewResourceConflictError("plan ID %s does not match existing value %s", request.PlanID, planID))
				return
			}

			context := &runtime.RawExtension{}

			ok, err = entry.Get(registry.Context, context)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing context", ErrUnexpected))
				return
			}

			newContext := &runtime.RawExtension{}
			if request.Context != nil {
				newContext = request.Context
			}

			if !reflect.DeepEqual(newContext, context) {
				jsonError(w, errors.NewResourceConflictError("request context %v does not match existing value %v", newContext, context))
				return
			}

			parameters := &runtime.RawExtension{}

			ok, err = entry.Get(registry.Parameters, parameters)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing parameters", ErrUnexpected))
				return
			}

			newParameters := &runtime.RawExtension{}
			if request.Parameters != nil {
				newParameters = request.Parameters
			}

			if !reflect.DeepEqual(newParameters, parameters) {
				jsonError(w, errors.NewResourceConflictError("request parameters %v do not match existing value %v", newParameters, parameters))
				return
			}

			status := http.StatusOK
			response := &api.CreateServiceInstanceResponse{}

			// There is some ambiguity in the specification, it's accepted if something is already
			// provisioning, or a conflict if it's already provisioning with different parameters,
			// but no mention is made if another operation is in flight e.g. update or deprovision.
			// We'll just call it a conflict.
			operationType, ok, err := entry.GetString(registry.Operation)
			if err != nil {
				jsonError(w, err)
				return
			}

			if ok {
				if operation.Type(operationType) != operation.TypeProvision {
					jsonError(w, errors.NewResourceConflictError("existing %v operation in progress", operationType))
					return
				}

				operationID, ok, err := entry.GetString(registry.OperationID)
				if err != nil {
					jsonError(w, err)
					return
				}

				if !ok {
					jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
					return
				}

				status = http.StatusAccepted
				response.Operation = operationID
			}

			dashboardURL, ok, err := entry.GetString(registry.DashboardURL)
			if err != nil {
				jsonError(w, err)
				return
			}

			if ok {
				response.DashboardURL = dashboardURL
			}

			JSONResponse(w, status, response)

			return
		}

		context := &runtime.RawExtension{}
		if request.Context != nil {
			context = request.Context
		}

		parameters := &runtime.RawExtension{}
		if request.Parameters != nil {
			parameters = request.Parameters
		}

		namespace, err := getNamespace(request.Context, configuration.Namespace)
		if err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.Namespace, namespace); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.InstanceID, instanceID); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.ServiceID, request.ServiceID); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.PlanID, request.PlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.Context, context); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.Parameters, parameters); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Commit(); err != nil {
			jsonError(w, err)
			return
		}

		glog.Infof("provisioning new service instance: %s", instanceID)

		// Create a provisioning engine, and perform synchronous tasks.  This also derives
		// things like the dashboard URL for the synchronous response.
		provisioner, err := provisioners.NewCreator(provisioners.ResourceTypeServiceInstance)
		if err != nil {
			jsonError(w, err)
			return
		}

		if err := provisioner.Prepare(entry); err != nil {
			jsonError(w, err)
			return
		}

		if err := operation.Start(entry, operation.TypeProvision); err != nil {
			jsonError(w, err)
			return
		}

		frozenEntry := entry.Clone()

		go provisioner.Run(entry)

		operationID, ok, err := frozenEntry.GetString(registry.OperationID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
			return
		}

		// Return a response to the client.
		response := &api.CreateServiceInstanceResponse{
			Operation: operationID,
		}

		dashboardURL, ok, err := frozenEntry.GetString(registry.DashboardURL)
		if err != nil {
			jsonError(w, err)
			return
		}

		if ok {
			response.DashboardURL = dashboardURL
		}

		JSONResponse(w, http.StatusAccepted, response)
	}
}

// handleReadServiceInstance allows a service instance to be read.
func handleReadServiceInstance(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		// Check if the instance exists.
		entry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, true)
		if err != nil {
			jsonError(w, err)
			return
		}

		// Not found, return a 404
		if !entry.Exists() {
			jsonError(w, errors.NewResourceNotFoundError("service instance does not exist"))
			return
		}

		// service_id is optional and provoded as a hint.
		serviceID, serviceIDProvided, err := maygetSingleParameter(r, "service_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		// plan_id is optional and provoded as a hint.
		planID, planIDProvided, err := maygetSingleParameter(r, "plan_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		serviceInstanceServiceID, ok, err := entry.GetString(registry.ServiceID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing service ID", ErrUnexpected))
			return
		}

		serviceInstancePlanID, ok, err := entry.GetString(registry.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
			return
		}

		if serviceIDProvided && serviceID != serviceInstanceServiceID {
			jsonError(w, errors.NewQueryError("specified service ID %s does not match %s", serviceID, serviceInstanceServiceID))
			return
		}

		if planIDProvided && planID != serviceInstancePlanID {
			jsonError(w, errors.NewQueryError("specified plan ID %s does not match %s", planID, serviceInstancePlanID))
			return
		}

		parameters := &runtime.RawExtension{}

		ok, err = entry.Get(registry.Parameters, parameters)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing parameters", ErrUnexpected))
			return
		}

		// If the instance does not exist or an operation is still in progress return
		// a 404.
		op, ok, err := entry.GetString(registry.Operation)
		if err != nil {
			jsonError(w, err)
			return
		}

		if ok {
			jsonError(w, errors.NewParameterError("%s operation in progress", op))
			return
		}

		response := &api.GetServiceInstanceResponse{
			ServiceID:  serviceInstanceServiceID,
			PlanID:     serviceInstancePlanID,
			Parameters: parameters,
		}
		JSONResponse(w, http.StatusOK, response)
	}
}

// handleUpdateServiceInstance allows a service instance to be modified.
func handleUpdateServiceInstance(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Ensure the client supports async operation.
		if err := asyncRequired(r); err != nil {
			jsonError(w, err)
			return
		}

		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonErrorUsable(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		// Parse the update request.
		request := &api.UpdateServiceInstanceRequest{}
		if err := jsonRequest(r, request); err != nil {
			jsonError(w, err)
			return
		}

		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		// Check if the instance already exists.
		// Check if the instance exists.
		entry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		// Not found, return a 404
		if !entry.Exists() {
			jsonError(w, errors.NewResourceNotFoundError("service instance does not exist"))
			return
		}

		// Get the plan from the registry, it is not guaranteed to be in the request.
		// Override with the request if specified.
		planID, ok, err := entry.GetString(registry.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
			return
		}

		newPlanID := planID
		if request.PlanID != "" {
			newPlanID = request.PlanID
		}

		// Check parameters.
		if err := validateServicePlan(config.Config(), request.ServiceID, newPlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := planUpdatable(config.Config(), request.ServiceID, planID, newPlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := validateParameters(config.Config(), request.ServiceID, planID, schemaTypeServiceInstance, schemaOperationUpdate, request.Parameters); err != nil {
			jsonErrorUsable(w, err)
			return
		}

		parameters := &runtime.RawExtension{}
		if request.Parameters != nil {
			parameters = request.Parameters
		}

		if err := entry.Set(registry.Parameters, parameters); err != nil {
			jsonError(w, err)
			return
		}

		updater, err := provisioners.NewUpdater(provisioners.ResourceTypeServiceInstance, request)
		if err != nil {
			jsonErrorUsable(w, err)
			return
		}

		if err := updater.Prepare(entry); err != nil {
			jsonErrorUsable(w, err)
			return
		}

		if err := operation.Start(entry, operation.TypeUpdate); err != nil {
			jsonError(w, err)
			return
		}

		frozenEntry := entry.Clone()

		go updater.Run(entry)

		operationID, ok, err := frozenEntry.GetString(registry.OperationID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
		}

		// Return a response to the client.
		response := &api.UpdateServiceInstanceResponse{
			Operation: operationID,
		}

		JSONResponse(w, http.StatusAccepted, response)
	}
}

// handleDeleteServiceInstance deletes a service instance.
func handleDeleteServiceInstance(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Ensure the client supports async operation.
		if err := asyncRequired(r); err != nil {
			jsonError(w, err)
			return
		}

		// Check parameters.
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		// Probably the wrong place for this...
		deleteDirectoryInstance(configuration.Namespace, instanceID)

		entry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !entry.Exists() {
			jsonError(w, errors.NewResourceGoneError("service instance does not exist"))
			return
		}

		serviceID, err := getSingleParameter(r, "service_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		planID, err := getSingleParameter(r, "plan_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		serviceInstanceServiceID, ok, err := entry.GetString(registry.ServiceID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing service ID", ErrUnexpected))
			return
		}

		serviceInstancePlanID, ok, err := entry.GetString(registry.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
			return
		}

		if serviceID != serviceInstanceServiceID {
			jsonError(w, errors.NewQueryError("specified service ID %s does not match %s", serviceID, serviceInstanceServiceID))
			return
		}

		if planID != serviceInstancePlanID {
			jsonError(w, errors.NewQueryError("specified plan ID %s does not match %s", planID, serviceInstancePlanID))
			return
		}

		deleter := provisioners.NewDeleter()

		// Start the delete operation in the background.
		if err := operation.Start(entry, operation.TypeDeprovision); err != nil {
			jsonError(w, err)
			return
		}

		go deleter.Run(entry)

		operationID, ok, err := entry.GetString(registry.OperationID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
		}

		response := &api.CreateServiceInstanceResponse{
			Operation: operationID,
		}
		JSONResponse(w, http.StatusAccepted, response)
	}
}

// handlePollServiceInstance polls a service instance operation for status.
func handlePollServiceInstance(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		entry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !entry.Exists() {
			JSONResponse(w, http.StatusGone, struct{}{})
			return
		}

		// service_id is optional and provoded as a hint.
		serviceID, serviceIDProvided, err := maygetSingleParameter(r, "service_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		// plan_id is optional and provided as a hint.
		planID, planIDProvided, err := maygetSingleParameter(r, "plan_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		// operation is optional, however the broker only implements asynchronous
		// operations at present, so require it unconditionally.
		operationID, err := getSingleParameter(r, "operation")
		if err != nil {
			jsonError(w, err)
			return
		}

		instanceServiceID, ok, err := entry.GetString(registry.ServiceID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
		}

		instancePlanID, ok, err := entry.GetString(registry.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
		}

		instanceOperationID, ok, err := entry.GetString(registry.OperationID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
		}

		// While not specified, we check that the provided service ID matches the one
		// we expect.  It may be indicative of a client error.
		if serviceIDProvided && serviceID != instanceServiceID {
			jsonError(w, errors.NewQueryError("provided service ID %s does not match %s", serviceID, instanceServiceID))
			return
		}

		// While not specified, we check that the provided plan ID matches the one
		// we expect.  It may be indicative of a client error.
		if planIDProvided && planID != instancePlanID {
			jsonError(w, errors.NewQueryError("provided plan ID %s does not match %s", planID, instancePlanID))
			return
		}

		if operationID != instanceOperationID {
			jsonError(w, errors.NewQueryError("provided operation %s does not match operation %s", operationID, instanceOperationID))
			return
		}

		operationStatus, ok, err := entry.GetString(registry.OperationStatus)
		if err != nil {
			jsonError(w, err)
			return
		}

		// If there is no status then the provisioning operation is still in progress (or has crashed...)
		if !ok {
			response := &api.PollServiceInstanceResponse{
				State:       api.PollStateInProgress,
				Description: "asynchronous provisioning in progress",
			}
			JSONResponse(w, http.StatusOK, response)

			return
		}

		// If the status isn't empty then we have encountered an error and need to report failure.
		if operationStatus != "" {
			if err := operation.End(entry); err != nil {
				jsonError(w, err)
				return
			}

			response := &api.PollServiceInstanceResponse{
				State:       api.PollStateFailed,
				Description: operationStatus,
			}
			JSONResponse(w, http.StatusOK, response)

			return
		}

		// Check the readiness of resources, an error is a genuine error whereas a condtition
		// unready error is expected and polling should continue.
		if err := provisioners.Ready(provisioners.ResourceTypeServiceInstance, entry, instanceServiceID, instancePlanID); err != nil {
			if provisioners.IsConditionUnreadyError(err) {
				response := &api.PollServiceInstanceResponse{
					State:       api.PollStateInProgress,
					Description: err.Error(),
				}
				JSONResponse(w, http.StatusOK, response)

				return
			}

			jsonError(w, err)

			return
		}

		// All checks have passed, instance successfully provisioned.
		if err := operation.End(entry); err != nil {
			jsonError(w, err)
			return
		}

		response := &api.PollServiceInstanceResponse{
			State: api.PollStateSucceeded,
		}
		JSONResponse(w, http.StatusOK, response)
	}
}

// handleCreateServiceBinding creates a binding to a service instance.
func handleCreateServiceBinding(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Parse the creation request.
		request := &api.CreateServiceBindingRequest{}
		if err := jsonRequest(r, request); err != nil {
			jsonError(w, err)
			return
		}

		// Check parameters.
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		bindingID := params.ByName("binding_id")
		if bindingID == "" {
			jsonError(w, fmt.Errorf("%w: request missing binding_id parameter", ErrUnexpected))
			return
		}

		if err := validateServicePlan(config.Config(), request.ServiceID, request.PlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := verifyBindable(config.Config(), request.ServiceID, request.PlanID); err != nil {
			jsonError(w, err)
			return
		}

		if err := validateParameters(config.Config(), request.ServiceID, request.PlanID, schemaTypeServiceBinding, schemaOperationCreate, request.Parameters); err != nil {
			jsonError(w, err)
			return
		}

		// Check if the service instance exists.
		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		instanceEntry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, true)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !instanceEntry.Exists() {
			jsonError(w, errors.NewParameterError("service instance %s not found", instanceID))
			return
		}

		// Check if the binding already exists.
		entry, err := registry.New(registry.ServiceBinding, dirent.Namespace, bindingID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		if entry.Exists() {
			// If the binding already exists either return 200 if provisioned or
			// a 202 if it is still provisioning, or a 409 if provisioned or
			// provisioning with different attributes.
			serviceID, ok, err := entry.GetString(registry.ServiceID)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing service ID", ErrUnexpected))
				return
			}

			if serviceID != request.ServiceID {
				jsonError(w, errors.NewResourceConflictError("service ID %s does not match existing value %s", request.ServiceID, serviceID))
				return
			}

			planID, ok, err := entry.GetString(registry.PlanID)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
				return
			}

			if planID != request.PlanID {
				jsonError(w, errors.NewResourceConflictError("plan ID %s does not match existing value %s", request.PlanID, planID))
				return
			}

			context := &runtime.RawExtension{}

			ok, err = entry.Get(registry.Context, context)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing context", ErrUnexpected))
				return
			}

			newContext := &runtime.RawExtension{}
			if request.Context != nil {
				newContext = request.Context
			}

			if !reflect.DeepEqual(newContext, context) {
				jsonError(w, errors.NewResourceConflictError("request context %v does not match existing value %v", newContext, context))
				return
			}

			parameters := &runtime.RawExtension{}

			ok, err = entry.Get(registry.Parameters, parameters)
			if err != nil {
				jsonError(w, err)
				return
			}

			if !ok {
				jsonError(w, fmt.Errorf("%w: unable to lookup existing parameters", ErrUnexpected))
				return
			}

			newParameters := &runtime.RawExtension{}
			if request.Parameters != nil {
				newParameters = request.Parameters
			}

			if !reflect.DeepEqual(newParameters, parameters) {
				jsonError(w, errors.NewResourceConflictError("request parameters %v do not match existing value %v", newParameters, parameters))
				return
			}

			status := http.StatusOK
			response := &api.CreateServiceBindingResponse{}

			// There is some ambiguity in the specification, it's accepted if something is already
			// provisioning, or a conflict if it's already provisioning with different parameters,
			// but no mention is made if another operation is in flight e.g. update or deprovision.
			// We'll just call it a conflict.
			operationType, ok, err := entry.GetString(registry.Operation)
			if err != nil {
				jsonError(w, err)
				return
			}

			if ok {
				if operation.Type(operationType) != operation.TypeProvision {
					jsonError(w, errors.NewResourceConflictError("existing %v operation in progress", operationType))
					return
				}

				operationID, ok, err := entry.GetString(registry.OperationID)
				if err != nil {
					jsonError(w, err)
					return
				}

				if !ok {
					jsonError(w, fmt.Errorf("%w: service instance missing operation ID", ErrUnexpected))
					return
				}

				response.Operation = operationID
			}

			JSONResponse(w, status, response)

			return
		}

		// The binding gets a copy of all service instance data, this could be used
		// to communicate TLS or other password information.  The context and parameters
		// are overridden buy those related to the binding.
		entry.Inherit(instanceEntry)

		context := &runtime.RawExtension{}
		if request.Context != nil {
			context = request.Context
		}

		parameters := &runtime.RawExtension{}
		if request.Parameters != nil {
			parameters = request.Parameters
		}

		if err := entry.Set(registry.BindingID, bindingID); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.Context, context); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Set(registry.Parameters, parameters); err != nil {
			jsonError(w, err)
			return
		}

		if err := entry.Commit(); err != nil {
			jsonError(w, err)
			return
		}

		glog.Infof("provisioning new service binding: %s", bindingID)

		// Create a provisioning engine, and perform synchronous tasks.  This also derives
		// things like the dashboard URL for the synchronous response.
		provisioner, err := provisioners.NewCreator(provisioners.ResourceTypeServiceBinding)
		if err != nil {
			jsonError(w, err)
			return
		}

		if err := provisioner.Prepare(entry); err != nil {
			jsonError(w, err)
			return
		}

		if err := operation.Start(entry, operation.TypeProvision); err != nil {
			jsonError(w, err)
			return
		}

		frozenEntry := entry.Clone()

		provisioner.Run(entry)

		operationStatus, ok, err := entry.GetString(registry.OperationStatus)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: expected operation status not found", ErrUnexpected))
			return
		}

		// Stop the operation to allow other things to happen now.
		if err := operation.End(entry); err != nil {
			jsonError(w, err)
			return
		}

		if operationStatus != "" {
			// Work needed: properly propagate the error type.
			jsonError(w, errors.NewConfigurationError(operationStatus))
			return
		}

		credentials := &runtime.RawExtension{}

		if _, err := frozenEntry.Get(registry.Credentials, credentials); err != nil {
			jsonError(w, err)
			return
		}

		response := &api.GetServiceBindingResponse{
			Credentials: credentials,
		}
		JSONResponse(w, http.StatusCreated, response)
	}
}

// handleDeleteServiceBinding deletes a service binding.
func handleDeleteServiceBinding(configuration *ServerConfiguration) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		instanceID := params.ByName("instance_id")
		if instanceID == "" {
			jsonError(w, fmt.Errorf("%w: request missing instance_id parameter", ErrUnexpected))
			return
		}

		dirent := getDirectoryInstance(configuration.Namespace, instanceID)

		instanceEntry, err := registry.New(registry.ServiceInstance, dirent.Namespace, instanceID, true)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !instanceEntry.Exists() {
			jsonError(w, errors.NewParameterError("service instance %s not found", instanceID))
			return
		}

		// Check parameters.
		bindingID := params.ByName("binding_id")
		if bindingID == "" {
			jsonError(w, fmt.Errorf("%w: request missing binding_id parameter", ErrUnexpected))
			return
		}

		entry, err := registry.New(registry.ServiceBinding, dirent.Namespace, bindingID, false)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !entry.Exists() {
			jsonError(w, errors.NewResourceGoneError("service instance does not exist"))
			return
		}

		serviceID, err := getSingleParameter(r, "service_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		planID, err := getSingleParameter(r, "plan_id")
		if err != nil {
			jsonError(w, err)
			return
		}

		serviceInstanceServiceID, ok, err := entry.GetString(registry.ServiceID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing service ID", ErrUnexpected))
			return
		}

		serviceInstancePlanID, ok, err := entry.GetString(registry.PlanID)
		if err != nil {
			jsonError(w, err)
			return
		}

		if !ok {
			jsonError(w, fmt.Errorf("%w: unable to lookup existing plan ID", ErrUnexpected))
			return
		}

		if serviceID != serviceInstanceServiceID {
			jsonError(w, errors.NewQueryError("specified service ID %s does not match %s", serviceID, serviceInstanceServiceID))
			return
		}

		if planID != serviceInstancePlanID {
			jsonError(w, errors.NewQueryError("specified plan ID %s does not match %s", planID, serviceInstancePlanID))
			return
		}

		deleter := provisioners.NewDeleter()

		deleter.Run(entry)

		response := &api.DeleteServiceBindingResponse{}
		JSONResponse(w, http.StatusOK, response)
	}
}
