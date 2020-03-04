package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/test/fixtures"
	"github.com/couchbase/service-broker/test/util"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestServiceInstanceCreateNotAynchronous tests that the service broker rejects service
// instance creation that isn't asynchronous.
func TestServiceInstanceCreateNotAynchronous(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	util.MustPutAndError(t, "/v2/service_instances/pinkiepie", http.StatusUnprocessableEntity, nil, api.ErrorAsyncRequired)
}

// TestServiceInstanceCreateIllegalBody tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, `illegal`, api.ErrorParameterError)
}

// TestServiceInstanceCreateIllegalConfiguration tests that the service broker handles
// misconfiguration of the service catalog gracefully.  On this occasion the default
// doesn't have any service offerings or plans defined.
func TestServiceInstanceCreateIllegalConfiguration(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestServiceInstanceCreateIllegalQuery tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true&%illegal", http.StatusBadRequest, req, api.ErrorQueryError)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = "illegal"
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = "illegal"
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceCreate tests that the service broker accepts a minimal
// service instance creation.
func TestServiceInstanceCreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
}

// TestServiceInstanceCreateWithSchema tests that the service broker accepts a
// minimal service instance creation with schema validation.
func TestServiceInstanceCreateWithSchema(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
}

// TestServiceInstanceCreateWithSchemaNoParameters tests that the service broker accepts a
// minimal service instance creation with schema validation and no parameters.
func TestServiceInstanceCreateWithSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
}

// TestServiceInstanceCreateSchemaValidationFail tests that the service broker rejects
// schema validation failure.
func TestServiceInstanceCreateSchemaValidationFail(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":"string"}`),
	}
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorValidationError)
}

// TestServiceInstanceCreateWithRequiredSchemaNoParameters tests that the service broker
// rejects a minimal service instance creation with required schema validation and no
// parameters.
func TestServiceInstanceCreateWithRequiredSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchemaRequired()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorValidationError)
}

// TestServiceInstanceCreateInProgress tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, before the operation has
// completed e.g. been acknowledged, should return a 202.
func TestServiceInstanceCreateInProgress(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
}

// TestServiceInstanceCreateInProgressMismatched tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceCreateInProgressMismatched(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, nil)
	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, nil)
}

// TestServiceInstanceCreateCompleted tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, after the operation has
// completed e.g. been acknowledged, should return a 200.
func TestServiceInstanceCreateCompleted(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusOK, req, nil)
}

// TestServiceInstanceCreateCompletedMismatched tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceCreateCompletedMismatched(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstancePollIllegalServiceID tests that the service ID supplied to a service
// instance polling operation must match that of the instance's service ID.
func TestServiceInstancePollIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	req.ServiceID = "illegal"
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the plan ID supplied to a service
// instance polling operation must match that of the instance's plan ID.
func TestServiceInstancePollIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	req.PlanID = "illegal"
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the operation ID supplied to a service
// instance polling operation must match that of the current operation.
func TestServiceInstancePollIllegalOperationID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	rsp.Operation = "illegal"
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteNotAsynchronous tests that a service instance delete must
// be an aysnchronous operation.
func TestServiceInstanceDeleteNotAsynchronous(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie", http.StatusUnprocessableEntity, api.ErrorAsyncRequired)
}

// TestServiceInstanceDeleteIllegalInstance tests  service instance deletion when there
// isn't a corresponding service instance.
func TestServiceInstanceDeleteIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusGone, api.ErrorResourceGone)
}

// TestServiceInstanceDelete tests that service instance deletion works.
func TestServiceInstanceDelete(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	deleteRsp := &api.CreateServiceInstanceResponse{}
	util.MustDelete(t, "/v2/service_instances/pinkiepie?"+util.DeleteServiceInstanceQuery(req), http.StatusAccepted, deleteRsp)

	poll = &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, deleteRsp), http.StatusOK, poll)
}

// TestServiceInstanceDeleteServiceIDRequired tests delete requests without service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	query := "accepts_incomplete=true&plan_id=" + req.PlanID
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query, http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteServiceIDInvalid tests delete requests with the wrong service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	query := "accepts_incomplete=true&service_id=illegal&plan_id=" + req.PlanID
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query, http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDRequired tests delete requests without plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	query := "accepts_incomplete=true&service_id=" + req.ServiceID
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query, http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDInvalid tests delete requests with the wrong plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	query := "accepts_incomplete=true&plan_id=illegal&service_id=" + req.ServiceID
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query, http.StatusBadRequest, api.ErrorQueryError)
}
