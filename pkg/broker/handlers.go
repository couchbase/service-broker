package broker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/provisioners"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/apimachinery/pkg/api/errors"
)

// handleReadyz is a handler for Kubernetes readiness checks.  It is less verbose than the
// other API calls as it's called significantly more often.
func handleReadyz(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	util.HTTPResponse(w, http.StatusOK)
}

// handleReadCatalog advertises the classes of service we offer, and specifc plans to
// implement those classes.
func handleReadCatalog(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	util.JSONResponse(w, http.StatusOK, config.Config().Spec.Catalog)
}

// handleCreateServiceInstance creates a service instance of a plan.
func handleCreateServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if !util.AsyncOnlyResponse(w, r) {
		return
	}

	// Parse the creation request.
	request := &api.CreateServiceInstanceRequest{}
	if !util.JSONRequest(w, r, request) {
		return
	}

	// Check parameters.
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, http.StatusBadRequest, fmt.Errorf("request missing instance_id parameter"))
		return
	}
	if err := util.ValidateParameters(config.Config(), request.ServiceID, request.PlanID, util.SchemaTypeServiceInstance, util.SchemaOperationCreate, request.Parameters); err != nil {
		util.JSONError(w, http.StatusBadRequest, err)
		return
	}

	// Check if the instance already exists.
	registryEntry, err := instanceRegistry.Get(registry.ServiceInstanceRegistryName(instanceID))
	if err != nil && !errors.IsNotFound(err) {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	if registryEntry != nil {
		// If the instance already exists either return 200 if provisioned or
		// a 202 if it is still provisioning, or a 409 if provisioned or
		// provisioning with different attributes.
		prevRequestRaw, err := registryEntry.Get(registry.ServiceInstanceRequestKey)
		if err != nil {
			util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("unable to get service instance request from registry: %v", err))
			return
		}
		prevRequest := &api.CreateServiceInstanceRequest{}
		if err := json.Unmarshal([]byte(prevRequestRaw), prevRequest); err != nil {
			util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal previous instance request: %v", err))
			return
		}
		if reflect.DeepEqual(request, prevRequest) {
			dashboardURL, _ := registryEntry.Get(registry.RegistryKeyDashboardURL)
			op, ok := operation.Get(instanceID)
			if !ok {
				response := &api.CreateServiceInstanceResponse{
					DashboardURL: dashboardURL,
				}
				util.JSONResponse(w, http.StatusOK, response)
				return
			}

			if op.Kind == operation.OperationKindServiceInstanceCreate {
				response := &api.CreateServiceInstanceResponse{
					DashboardURL: dashboardURL,
					Operation:    op.ID,
				}
				util.JSONResponse(w, http.StatusAccepted, response)
				return
			}
		}
		util.JSONError(w, http.StatusConflict, fmt.Errorf("request conflicts with existing service instance"))
		return
	}

	// Create a registry entry in the broker's namespace.  We cannot use the context's namespace
	// as when we receive DELETE requests for example this context is not available and we don't
	// know where to look.
	registryEntry, err = instanceRegistry.New(registry.ServiceInstanceRegistryName(instanceID))
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to create registry entry for request: %v", err))
		return
	}

	// Save the raw request in the registry, it is required for other handler logic.
	requestJSON, err := json.Marshal(request)
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to marshal instance data: %v", err))
		return
	}
	if err := registryEntry.Set(registry.ServiceInstanceRequestKey, string(requestJSON)); err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to set registry entry value %s: %v", registry.ServiceInstanceRequestKey, err))
		return
	}
	if err := registryEntry.Set(registry.ServiceOfferingKey, request.ServiceID); err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to set registry entry value %s: %v", registry.ServiceOfferingKey, err))
		return
	}
	if err := registryEntry.Set(registry.ServicePlanKey, request.PlanID); err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to set registry entry value %s: %v", registry.ServicePlanKey, err))
		return
	}

	glog.Infof("provisioning new service instance: %s", string(requestJSON))

	// Create a provisioning engine, and perform synchronous tasks.  This also derives
	// things like the dashboard URL for the synchronous response.
	provisioner, err := provisioners.NewServiceInstanceCreator(instanceRegistry, instanceID, request)
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to create provisioner: %v", err))
		return
	}
	if err := provisioner.PrepareServiceInstance(); err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to prepare service instance: %v", err))
		return
	}

	// Start the provisioning process in the background.
	op := operation.New(operation.OperationKindServiceInstanceCreate, instanceID)
	go op.Run(provisioner)

	// Return a response to the client.
	dashboardURL, _ := registryEntry.Get(registry.RegistryKeyDashboardURL)
	response := &api.CreateServiceInstanceResponse{
		DashboardURL: dashboardURL,
		Operation:    op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleReadServiceInstance
func handleReadServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, http.StatusBadRequest, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	registryEntry, err := instanceRegistry.Get(registry.ServiceInstanceRegistryName(instanceID))
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	// If the instance does not exist or an operation is still in progress return
	// a 404.
	_, ok := operation.Get(instanceID)
	if errors.IsNotFound(err) || ok {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	requestJSON, err := registryEntry.Get(registry.ServiceInstanceRequestKey)
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	request := &api.CreateServiceInstanceRequest{}
	if err := json.Unmarshal([]byte(requestJSON), request); err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal instance request: %v", err))
		return
	}

	dashboardURL, _ := registryEntry.Get(registry.RegistryKeyDashboardURL)
	response := &api.GetServiceInstanceResponse{
		ServiceID:    request.ServiceID,
		PlanID:       request.PlanID,
		DashboardURL: dashboardURL,
		Parameters:   request.Parameters,
	}
	util.JSONResponse(w, http.StatusOK, response)
}

// handleUpdateServiceInstance
func handleUpdateServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if !util.AsyncOnlyResponse(w, r) {
		return
	}

	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONErrorUsable(w, http.StatusBadRequest, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	// Check if the instance already exists.
	registryEntry, err := instanceRegistry.Get(registry.ServiceInstanceRegistryName(instanceID))
	if err != nil && !errors.IsNotFound(err) {
		util.JSONErrorUsable(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	// Not found, return a 404
	if registryEntry == nil {
		util.JSONError(w, http.StatusNotFound, fmt.Errorf("service instance does not exist"))
		return
	}

	// Get the plan from the registry, it is not guaranyeed to be in the rquest.
	planID, err := registryEntry.Get(registry.ServicePlanKey)
	if err != nil {
		util.JSONError(w, http.StatusNotFound, fmt.Errorf("unable to lookup service instance plan ID: %v", err))
		return
	}

	// Parse the update request.
	request := &api.UpdateServiceInstanceRequest{}
	if !util.JSONRequest(w, r, request) {
		return
	}

	// Check parameters.
	if err := util.ValidateParameters(config.Config(), request.ServiceID, planID, util.SchemaTypeServiceInstance, util.SchemaOperationUpdate, request.Parameters); err != nil {
		util.JSONErrorUsable(w, http.StatusBadRequest, err)
		return
	}

	updater, err := provisioners.NewServiceInstanceUpdater(instanceRegistry, instanceID, request)
	if err != nil {
		util.JSONErrorUsable(w, http.StatusInternalServerError, err)
		return
	}
	if err := updater.PrepareResources(); err != nil {
		util.JSONErrorUsable(w, http.StatusInternalServerError, err)
		return
	}

	// Start the update operation in the background.
	op := operation.New(operation.OperationKindServiceInstanceUpdate, instanceID)
	go op.Run(updater)

	// Return a response to the client.
	dashboardURL, _ := registryEntry.Get(registry.RegistryKeyDashboardURL)
	response := &api.UpdateServiceInstanceResponse{
		DashboardURL: dashboardURL,
		Operation:    op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleDeleteServiceInstance
func handleDeleteServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if !util.AsyncOnlyResponse(w, r) {
		return
	}

	// Check parameters.
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, http.StatusBadRequest, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	if _, err := instanceRegistry.Get(registry.ServiceInstanceRegistryName(instanceID)); err != nil {
		if errors.IsNotFound(err) {
			util.JSONError(w, http.StatusGone, fmt.Errorf("service instance does not exist"))
			return
		}
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup resigstry instance: %v", err))
		return
	}

	deleter := provisioners.NewServiceInstanceDeleter(instanceRegistry, instanceID)

	// Start the delete operation in the background.
	op := operation.New(operation.OperationKindServiceInstanceDelete, instanceID)
	go op.Run(deleter)

	response := &api.CreateServiceInstanceResponse{
		Operation: op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleReadServiceInstanceStatus
func handleReadServiceInstanceStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	instanceID := params.ByName("instance_id")

	// TODO: Check all parameters

	op, ok := operation.Get(instanceID)
	if !ok {
		// While we should return an error here, the service catalog doesn't like getting
		// a non-yay or nay response.
		response := &api.PollServiceInstanceResponse{
			State: api.PollStateSucceeded,
		}
		util.JSONResponse(w, http.StatusOK, response)
		return
	}

	// Poll the provisioner process for status updates.
	var status string
	var description string
	select {
	case err := <-op.Status:
		// Free memory.  Is it safer just to garbage collect?  Yes.
		operation.Delete(instanceID)
		if err != nil {
			status = api.PollStateFailed
			description = err.Error()
			glog.Error(err)
			break
		}
		status = api.PollStateSucceeded
	default:
		status = api.PollStateInProgress
	}

	// Return a response to the client.
	response := &api.PollServiceInstanceResponse{
		State:       status,
		Description: description,
	}
	util.JSONResponse(w, http.StatusOK, response)
}

// handleCreateServiceBinding
func handleCreateServiceBinding(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if !util.AsyncOnlyResponse(w, r) {
		return
	}

	// Check request parameters.
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, http.StatusBadRequest, fmt.Errorf("request missing instance_id parameter"))
		return
	}
	bindingID := params.ByName("binding_id")
	if bindingID == "" {
		util.JSONError(w, http.StatusBadRequest, fmt.Errorf("request missing binding_id parameter"))
		return
	}

	// Parse and validate the request.
	request := &api.CreateServiceBindingRequest{}
	if !util.JSONRequest(w, r, request) {
		return
	}
	if err := util.ValidateParameters(config.Config(), request.ServiceID, request.PlanID, util.SchemaTypeServiceBinding, util.SchemaOperationCreate, request.Parameters); err != nil {
		util.JSONError(w, http.StatusBadRequest, err)
		return
	}

	// Check for an existing binding.
	registryEntry, err := instanceRegistry.Get(registry.ServiceBindingRegistryName(instanceID, bindingID))
	if err != nil && !errors.IsNotFound(err) {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to lookup registry entry: %v", err))
		return
	}

	// If the binding alread exists with the same parmeters return 202.  Return
	// a 409 if they differ.
	if registryEntry != nil {
		return
	}

	_, err = instanceRegistry.New(registry.ServiceBindingRegistryName(instanceID, bindingID))
	if err != nil {
		util.JSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to create registry entry for request: %v", err))
		return
	}

	creator := provisioners.NewServiceBindingCreator(instanceRegistry, instanceID, bindingID)

	// Start the provisioning process in the background.
	op := operation.New(operation.OperationKindServiceInstanceCreate, instanceID)
	go op.Run(creator)

	// Respond the operation ID to the client to start polling.
	response := &api.CreateServiceBindingResponse{
		Operation: op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleReadServiceBinding
func handleReadServiceBinding(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
}

// handleUpdateServiceBinding
func handleUpdateServiceBinding(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
}

// handleDeleteServiceBinding
func handleDeleteServiceBinding(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
}

// handleReadServiceBindingStatus
func handleReadServiceBindingStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
}
