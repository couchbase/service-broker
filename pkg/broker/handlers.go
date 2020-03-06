package broker

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/provisioners"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/apimachinery/pkg/runtime"
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
	if err := util.AsyncRequired(r); err != nil {
		util.JSONError(w, err)
		return
	}

	// Parse the creation request.
	request := &api.CreateServiceInstanceRequest{}
	if err := util.JSONRequest(r, request); err != nil {
		util.JSONError(w, err)
		return
	}

	// Check parameters.
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	if err := util.ValidateServicePlan(config.Config(), request.ServiceID, request.PlanID); err != nil {
		util.JSONError(w, err)
		return
	}

	if err := util.ValidateParameters(config.Config(), request.ServiceID, request.PlanID, util.SchemaTypeServiceInstance, util.SchemaOperationCreate, request.Parameters); err != nil {
		util.JSONError(w, err)
		return
	}

	// Check if the instance already exists.
	entry, err := registry.Instance(instanceID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	if entry.Exists() {
		// If the instance already exists either return 200 if provisioned or
		// a 202 if it is still provisioning, or a 409 if provisioned or
		// provisioning with different attributes.
		serviceID, ok := entry.Get(registry.ServiceID)
		if !ok {
			util.JSONError(w, fmt.Errorf("unable to lookup existing service ID"))
			return
		}

		if serviceID != request.ServiceID {
			util.JSONError(w, errors.NewResourceConflictError("service ID %s does not match existing value %s", request.ServiceID, serviceID))
			return
		}

		planID, ok := entry.Get(registry.PlanID)
		if !ok {
			util.JSONError(w, fmt.Errorf("unable to lookup existing plan ID"))
			return
		}

		if planID != request.PlanID {
			util.JSONError(w, errors.NewResourceConflictError("plan ID %s does not match existing value %s", request.PlanID, planID))
			return
		}

		context := &runtime.RawExtension{}

		ok, err := entry.GetJSON(registry.Context, context)
		if err != nil {
			util.JSONError(w, err)
			return
		}

		if !ok {
			util.JSONError(w, fmt.Errorf("unable to lookup existing context"))
			return
		}

		newContext := &runtime.RawExtension{}
		if request.Context != nil {
			newContext = request.Context
		}

		if !reflect.DeepEqual(newContext, context) {
			util.JSONError(w, errors.NewResourceConflictError("request context %v does not match existing value %v", newContext, context))
			return
		}

		parameters := &runtime.RawExtension{}

		ok, err = entry.GetJSON(registry.Parameters, parameters)
		if err != nil {
			util.JSONError(w, err)
			return
		}

		if !ok {
			util.JSONError(w, fmt.Errorf("unable to lookup existing parameters"))
			return
		}

		newParameters := &runtime.RawExtension{}
		if request.Parameters != nil {
			newParameters = request.Parameters
		}

		if !reflect.DeepEqual(newParameters, parameters) {
			util.JSONError(w, errors.NewResourceConflictError("request parameters %v do not match existing value %v", newParameters, parameters))
			return
		}

		status := http.StatusOK
		response := &api.CreateServiceInstanceResponse{}

		if op, ok := operation.Get(instanceID); ok {
			status = http.StatusAccepted
			response.Operation = op.ID
		}

		util.JSONResponse(w, status, response)

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

	entry.Set(registry.ServiceID, request.ServiceID)
	entry.Set(registry.PlanID, request.PlanID)

	if err := entry.SetJSON(registry.Context, context); err != nil {
		util.JSONError(w, err)
		return
	}

	if err := entry.SetJSON(registry.Parameters, parameters); err != nil {
		util.JSONError(w, err)
		return
	}

	if err := entry.Commit(); err != nil {
		util.JSONError(w, err)
		return
	}

	glog.Infof("provisioning new service instance: %s", instanceID)

	// Create a provisioning engine, and perform synchronous tasks.  This also derives
	// things like the dashboard URL for the synchronous response.
	provisioner, err := provisioners.NewServiceInstanceCreator(entry, instanceID, request)
	if err != nil {
		util.JSONError(w, fmt.Errorf("failed to create provisioner: %v", err))
		return
	}

	if err := provisioner.PrepareServiceInstance(); err != nil {
		util.JSONError(w, fmt.Errorf("failed to prepare service instance: %v", err))
		return
	}

	// Start the provisioning process in the background.
	op, err := operation.New(operation.TypeServiceInstanceCreate, instanceID, request.ServiceID, request.PlanID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	go op.Run(provisioner)

	// Return a response to the client.
	response := &api.CreateServiceInstanceResponse{
		Operation: op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleReadServiceInstance
func handleReadServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	// Check if the instance exists.
	entry, err := registry.Instance(instanceID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// Not found, return a 404
	if !entry.Exists() {
		util.JSONError(w, errors.NewResourceNotFoundError("service instance does not exist"))
		return
	}

	// service_id is optional and provoded as a hint.
	serviceID, serviceIDProvided, err := util.MayGetSingleParameter(r, "service_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// plan_id is optional and provoded as a hint.
	planID, planIDProvided, err := util.MayGetSingleParameter(r, "plan_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	serviceInstanceServiceID, ok := entry.Get(registry.ServiceID)
	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing service ID"))
		return
	}

	serviceInstancePlanID, ok := entry.Get(registry.PlanID)
	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing plan ID"))
		return
	}

	if serviceIDProvided && serviceID != serviceInstanceServiceID {
		util.JSONError(w, errors.NewQueryError("specified service ID %s does not match %s", serviceID, serviceInstanceServiceID))
		return
	}

	if planIDProvided && planID != serviceInstancePlanID {
		util.JSONError(w, errors.NewQueryError("specified plan ID %s does not match %s", planID, serviceInstancePlanID))
		return
	}

	parameters := &runtime.RawExtension{}

	ok, err = entry.GetJSON(registry.Parameters, parameters)
	if err != nil {
		util.JSONError(w, err)
	}

	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing parameters"))
		return
	}

	// If the instance does not exist or an operation is still in progress return
	// a 404.
	if _, ok := operation.Get(instanceID); ok {
		util.JSONError(w, errors.NewParameterError("operation in progress"))
		return
	}

	response := &api.GetServiceInstanceResponse{
		ServiceID:  serviceInstanceServiceID,
		PlanID:     serviceInstancePlanID,
		Parameters: parameters,
	}
	util.JSONResponse(w, http.StatusOK, response)
}

// handleUpdateServiceInstance
func handleUpdateServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if err := util.AsyncRequired(r); err != nil {
		util.JSONError(w, err)
		return
	}

	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONErrorUsable(w, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	// Parse the update request.
	request := &api.UpdateServiceInstanceRequest{}
	if err := util.JSONRequest(r, request); err != nil {
		util.JSONError(w, err)
		return
	}

	// Check if the instance already exists.
	// Check if the instance exists.
	entry, err := registry.Instance(instanceID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// Not found, return a 404
	if !entry.Exists() {
		util.JSONError(w, errors.NewResourceNotFoundError("service instance does not exist"))
		return
	}

	// Get the plan from the registry, it is not guaranteed to be in the request.
	planID, ok := entry.Get(registry.PlanID)
	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing plan ID"))
		return
	}

	if request.PlanID != "" {
		planID = request.PlanID
	}

	// Check parameters.
	if err := util.ValidateServicePlan(config.Config(), request.ServiceID, planID); err != nil {
		util.JSONError(w, err)
		return
	}

	if err := util.ValidateParameters(config.Config(), request.ServiceID, planID, util.SchemaTypeServiceInstance, util.SchemaOperationUpdate, request.Parameters); err != nil {
		util.JSONErrorUsable(w, err)
		return
	}

	updater, err := provisioners.NewServiceInstanceUpdater(entry, instanceID, request)
	if err != nil {
		util.JSONErrorUsable(w, err)
		return
	}

	if err := updater.PrepareResources(); err != nil {
		util.JSONErrorUsable(w, err)
		return
	}

	// Start the update operation in the background.
	op, err := operation.New(operation.TypeServiceInstanceUpdate, instanceID, request.ServiceID, planID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	go op.Run(updater)

	// Return a response to the client.
	response := &api.UpdateServiceInstanceResponse{
		Operation: op.ID,
	}

	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleDeleteServiceInstance
func handleDeleteServiceInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Ensure the client supports async operation.
	if err := util.AsyncRequired(r); err != nil {
		util.JSONError(w, err)
		return
	}

	// Check parameters.
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	entry, err := registry.Instance(instanceID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	if !entry.Exists() {
		util.JSONError(w, errors.NewResourceGoneError("service instance does not exist"))
		return
	}

	serviceID, err := util.GetSingleParameter(r, "service_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	planID, err := util.GetSingleParameter(r, "plan_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	serviceInstanceServiceID, ok := entry.Get(registry.ServiceID)
	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing service ID"))
		return
	}

	serviceInstancePlanID, ok := entry.Get(registry.PlanID)
	if !ok {
		util.JSONError(w, fmt.Errorf("unable to lookup existing plan ID"))
		return
	}

	if serviceID != serviceInstanceServiceID {
		util.JSONError(w, errors.NewQueryError("specified service ID %s does not match %s", serviceID, serviceInstanceServiceID))
		return
	}

	if planID != serviceInstancePlanID {
		util.JSONError(w, errors.NewQueryError("specified plan ID %s does not match %s", planID, serviceInstancePlanID))
		return
	}

	deleter := provisioners.NewServiceInstanceDeleter(entry, instanceID)

	// Start the delete operation in the background.
	op, err := operation.New(operation.TypeServiceInstanceDelete, instanceID, serviceID, planID)
	if err != nil {
		util.JSONError(w, err)
		return
	}

	go op.Run(deleter)

	response := &api.CreateServiceInstanceResponse{
		Operation: op.ID,
	}
	util.JSONResponse(w, http.StatusAccepted, response)
}

// handleReadServiceInstanceStatus
func handleReadServiceInstanceStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	instanceID := params.ByName("instance_id")
	if instanceID == "" {
		util.JSONError(w, fmt.Errorf("request missing instance_id parameter"))
		return
	}

	op, ok := operation.Get(instanceID)
	if !ok {
		// The operation should be persistent, hence this hack.
		// While we should return an error here, the service catalog doesn't like getting
		// a non-yay or nay response.
		response := &api.PollServiceInstanceResponse{
			State: api.PollStateSucceeded,
		}

		util.JSONResponse(w, http.StatusOK, response)

		return
	}

	// service_id is optional and provoded as a hint.
	serviceID, serviceIDProvided, err := util.MayGetSingleParameter(r, "service_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// plan_id is optional and provided as a hint.
	planID, planIDProvided, err := util.MayGetSingleParameter(r, "plan_id")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// operation is optional, however the broker only implements asynchronous
	// operations at present, so require it unconditionally.
	operationID, err := util.GetSingleParameter(r, "operation")
	if err != nil {
		util.JSONError(w, err)
		return
	}

	// While not specified, we check that the provided service ID matches the one
	// we expect.  It may be indicative of a client error.
	if serviceIDProvided && serviceID != op.ServiceID {
		util.JSONError(w, errors.NewQueryError("provided service ID %s does not match %s", serviceID, op.ServiceID))
		return
	}

	// While not specified, we check that the provided plan ID matches the one
	// we expect.  It may be indicative of a client error.
	if planIDProvided && planID != op.PlanID {
		util.JSONError(w, errors.NewQueryError("provided plan ID %s does not match %s", planID, op.PlanID))
		return
	}

	if operationID != op.ID {
		util.JSONError(w, errors.NewQueryError("provided operation %s does not match operation %s", operationID, op.ID))
		return
	}

	// status is the API state of the operation.
	var status api.PollState

	// description is a description of why the operation is in that state.
	var description string

	// Poll the provisioner process for status updates.
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
