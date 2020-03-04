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

	util.MustPutWithError(t, "/v2/service_instances/pinkiepie", ``, http.StatusUnprocessableEntity, api.ErrorAsyncRequired)
}

// TestServiceInstanceCreateIllegalBody tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", `illegal`, http.StatusBadRequest, api.ErrorParameterError)
}

// TestServiceInstanceCreateIllegalConfiguration tests that the service broker handles
// misconfiguration of the service catalog gracefully.  On this occasion the default
// doesn't have any service offerings or plans defined.
func TestServiceInstanceCreateIllegalConfiguration(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorConfigurationError)
}

// TestServiceInstanceCreateIllegalQuery tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true&%illegal", req, http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = "illegal"
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorParameterError)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = "illegal"
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorParameterError)
}

// TestServiceInstanceCreate tests that the service broker accepts a minimal
// service instance creation.
func TestServiceInstanceCreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
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
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
}

// TestServiceInstanceCreateWithSchemaNoParameters tests that the service broker accepts a
// minimal service instance creation with schema validation and no parameters.
func TestServiceInstanceCreateWithSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
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
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorValidationError)
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
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorValidationError)
}

// TestServiceInstanceCreateInProgress tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, before the operation has
// completed e.g. been acknowledged, should return a 202.
func TestServiceInstanceCreateInProgress(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
}

// TestServiceInstanceCreateInProgressMismatched tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceCreateInProgressMismatched(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusConflict)
}

// TestServiceInstanceCreateCompleted tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, after the operation has
// completed e.g. been acknowledged, should return a 200.
func TestServiceInstanceCreateCompleted(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPutWithResponse(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusOK)
}

// TestServiceInstanceCreateCompletedMismatched tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceCreateCompletedMismatched(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPutWithResponse(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted, rsp)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusOK, poll)

	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusConflict)
}

// TestServiceInstancePollIllegalServiceID tests that the service ID supplied to a service
// instance polling operation must match that of the instance's service ID.
func TestServiceInstancePollIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPutWithResponse(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted, rsp)

	req.ServiceID = "illegal"
	util.MustGetWithError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the plan ID supplied to a service
// instance polling operation must match that of the instance's plan ID.
func TestServiceInstancePollIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPutWithResponse(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted, rsp)

	req.PlanID = "illegal"
	util.MustGetWithError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the operation ID supplied to a service
// instance polling operation must match that of the current operation.
func TestServiceInstancePollIllegalOperationID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := &api.CreateServiceInstanceResponse{}
	util.MustPutWithResponse(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted, rsp)

	rsp.Operation = "illegal"
	util.MustGetWithError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp), http.StatusBadRequest, api.ErrorQueryError)
}
