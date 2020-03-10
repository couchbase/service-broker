package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/test/fixtures"
	"github.com/couchbase/service-broker/test/util"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestServiceInstanceCreate tests that the service broker accepts a minimal
// service instance creation.
func TestServiceInstanceCreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

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

	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true&%illegal", http.StatusBadRequest, nil, api.ErrorQueryError)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = fixtures.IllegalID
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = fixtures.IllegalID
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
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
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstanceCreateWithSchemaNoParameters tests that the service broker accepts a
// minimal service instance creation with schema validation and no parameters.
func TestServiceInstanceCreateWithSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstanceCreateWithSchemaInvalid tests that the service broker rejects
// schema validation failure.
func TestServiceInstanceCreateWithSchemaInvalid(t *testing.T) {
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

// TestServiceInstancePoll tests polling a completed service instance creation
// is ok
func TestServiceInstancePoll(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstancePollServiceIDOptional tests that the service ID supplied to a service
// instance polling operation is optional.
func TestServiceInstancePollServiceIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	query.Del("service_id")
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+query.Encode(), http.StatusOK, nil)
}

// TestServiceInstancePollPlanIDOptional tests that the plan ID supplied to a service
// instance polling operation is optional.
func TestServiceInstancePollPlanIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	query.Del("plan_id")
	util.MustGet(t, "/v2/service_instances/pinkiepie/last_operation?"+query.Encode(), http.StatusOK, nil)
}

// TestServiceInstancePollIllegalQuery tests polling a completed service instance creation
// with a malformed query results in a bad request error.
func TestServiceInstancePollIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+query.Encode()+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the service ID supplied to a service
// instance polling operation must match that of the instance's service ID.
func TestServiceInstancePollIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	req.ServiceID = fixtures.IllegalID
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp).Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the plan ID supplied to a service
// instance polling operation must match that of the instance's plan ID.
func TestServiceInstancePollIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.IllegalID
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp).Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the operation ID supplied to a service
// instance polling operation must match that of the current operation.
func TestServiceInstancePollIllegalOperationID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	rsp.Operation = "illegal"
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie/last_operation?"+util.PollServiceInstanceQuery(req, rsp).Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceRecreateWhileInProgress tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, before the operation has
// completed e.g. been acknowledged, should return a 202.
func TestServiceInstanceRecreateWhileInProgress(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)
	util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstanceRecreateWhileInProgressMismatched tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceRecreateWhileInProgressMismatched(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)
	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, nil)
}

// TestServiceInstanceRecreateAfterCompletion tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, after the operation has
// completed e.g. been acknowledged, should return a 200.
func TestServiceInstanceRecreateAfterCompletion(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusOK, req, nil)
}

// TestServiceInstanceRecreateAfterCompletionMismatchedPlanID tests the behaviour of multiple creation
// requests for the same service instance with different request parameters, before the
// operation has completed e.g. been acknowledged, should return a 409.
func TestServiceInstanceRecreateAfterCompletionMismatchedPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstanceRecreateAfterCompletionMismatchedNoContext tests recreation of a service
// instance where the first was created with no context, and the second was.
func TestServiceInstanceRecreateAfterCompletionMismatchedNoContext(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.Context = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstanceRecreateAfterCompletionMismatchedWithContext tests recreation of a service
// instance where the first was created with a context, and the second wasn't.
func TestServiceInstanceRecreateAfterCompletionMismatchedWithContext(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Context = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.Context = nil
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstanceRecreateAfterCompletionMismatchedNoParameters tests recreation of a service
// instance where the first was created with no parameters, and the second was.
func TestServiceInstanceRecreateAfterCompletionMismatchedNoParameters(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstanceRecreateAfterCompletionMismatchedWithParameters tests recreation of a service
// instance where the first was created with parameters, and the second wasn't.
func TestServiceInstanceRecreateAfterCompletionMismatchedWithParameters(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.Parameters = nil
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusConflict, req, api.ErrorResourceConflict)
}

// TestServiceInstanceDelete tests that service instance deletion works.
func TestServiceInstanceDelete(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustDeleteServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
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

// TestServiceInstanceDeleteIllegalQuery tests that a malformed query raises a bad request
// with a query error.
func TestServiceInstanceDeleteIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode()+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteServiceIDRequired tests delete requests without service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Del("service_id")
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteServiceIDInvalid tests delete requests with the wrong service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Set("service_id", "illegal")
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDRequired tests delete requests without plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Del("plan_id")
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDInvalid tests delete requests with the wrong plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Set("plan_id", "illegal")
	util.MustDeleteAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteAndRecreate tests that persistent data isn't left lying about
// and we can recreate an instance.
func TestServiceInstanceDeleteAndRecreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustDeleteServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstanceRead tests that we can read an existing service instance.
func TestServiceInstanceRead(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustGet(t, "/v2/service_instances/pinkiepie?"+util.ReadServiceInstanceQuery(req).Encode(), http.StatusOK, nil)
}

// TestServiceInstanceReadIllegalQuery tests that a malformed query raises a bad request
// with a query error.
func TestServiceInstanceReadIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode()+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadServiceIDOptional tests that we can read an existing service instance
// and the service_id parameter is optional.
func TestServiceInstanceReadServiceIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Del("service_id")
	util.MustGet(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusOK, nil)
}

// TestServiceInstanceReadPlanIDOptional tests that we can read an existing service instance
// and the plan_id parameter is optional.
func TestServiceInstanceReadPlanIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Del("plan_id")
	util.MustGet(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusOK, nil)
}

// TestServiceInstanceReadServiceIDInvalid tests that we can read an existing service instance
// and the service_id parameter is llegal.
func TestServiceInstanceReadServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Set("service_id", fixtures.IllegalID)
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadPlanIDIllegal tests that we can read an existing service instance
// and the plan_id parameter is illegal.
func TestServiceInstanceReadPlanIDIllegal(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Set("plan_id", fixtures.IllegalID)
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie?"+query.Encode(), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadIllegalInstance tests that a read on an illegal service
// instance is rejected.
func TestServiceInstanceReadIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustGetAndError(t, "/v2/service_instances/pinkiepie?"+util.ReadServiceInstanceQuery(req).Encode(), http.StatusNotFound, api.ErrorResourceNotFound)
}

// TestServiceInstanceUpdate tests a service instance can be updated.
func TestServiceInstanceUpdate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	update := fixtures.BasicServiceInstanceUpdateRequest()
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)
}

// TestServiceInstanceUpdateAsyncNotAsynchronous tests that update operations must
// be asynchronous.
func TestServiceInstanceUpdateAsyncNotAsynchronous(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie", http.StatusUnprocessableEntity, nil, api.ErrorAsyncRequired)
}

// TestServiceInstanceUpdateIllegalBody tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, `illegal`, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalQuery tests that an illegal query raises a bad
// request and query error.
func TestServiceInstanceUpdateIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?%illegal", http.StatusBadRequest, nil, api.ErrorQueryError)
}

// TestServiceInstanceUpdateIllegalServiceID tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.ServiceID = fixtures.IllegalID
	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalPlanID tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.IllegalID
	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalInstance tests that an illegal instance raises a not
// found and resource not found error.
func TestServiceInstanceUpdateIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	update := fixtures.BasicServiceInstanceUpdateRequest()
	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusNotFound, update, api.ErrorResourceNotFound)
}

// TestServiceInstanceUpdateWithSchema tests that schema validation works.
func TestServiceInstanceUpdateWithSchema(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	update := fixtures.BasicServiceInstanceUpdateRequest()
	update.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)
}

// TestServiceInstanceUpdateWithSchemaNoParameters tests that schema validation
// is optional.
func TestServiceInstanceUpdateWithSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	update := fixtures.BasicServiceInstanceUpdateRequest()
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)
}

// TestServiceInstanceUpdateWithSchemaInvalid tests that schema validation rejects
// invalid parameters,
func TestServiceInstanceUpdateWithSchemaInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	update := fixtures.BasicServiceInstanceUpdateRequest()
	update.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":"string"}`),
	}
	util.MustPatchAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, update, api.ErrorValidationError)
}
