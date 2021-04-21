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

package unit_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/test/unit/fixtures"
	"github.com/couchbase/service-broker/test/unit/util"

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

// TestServiceInstanceCreateSingleton tests that the service broker accepts a
// minimal service instance creation and allows multiple instances sharing  a
// singleton resource.
func TestServiceInstanceCreateSingleton(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.AlternateServiceInstanceName, req)
}

// TestServiceInstanceCreateNotAynchronous tests that the service broker rejects service
// instance creation that isn't asynchronous.
func TestServiceInstanceCreateNotAynchronous(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, nil), http.StatusUnprocessableEntity, nil, api.ErrorAsyncRequired)
}

// TestServiceInstanceCreateIllegalBody tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, `illegal`, api.ErrorParameterError)
}

// TestServiceInstanceCreateIllegalConfiguration tests that the service broker handles
// misconfiguration of the service catalog gracefully.  On this occasion the default
// doesn't have any configuration bindings defined.
func TestServiceInstanceCreateIllegalConfiguration(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings = nil
	util.MustReplaceBrokerConfigWithInvalidCondition(t, clients, configuration)
}

// TestServiceInstanceCreateIllegalQuery tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery())+"&%illegal", http.StatusBadRequest, nil, api.ErrorQueryError)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = fixtures.IllegalID
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = fixtures.IllegalID
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorParameterError)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorValidationError)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorValidationError)
}

// TestServiceInstancePoll tests polling a completed service instance creation
// is ok.
func TestServiceInstancePoll(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstancePollWithReadiness tests that a configuration with readiness checks
// responds correctly.
func TestServiceInstancePollWithReadiness(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfigurationWithReadiness())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	poll := &api.PollServiceInstanceResponse{}
	util.MustGet(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, util.PollServiceInstanceQuery(nil, rsp)), http.StatusOK, poll)
	util.Assert(t, poll.State == api.PollStateInProgress)

	fixtures.MustSetFixtureField(t, clients, fixtures.BasicResourceStatus(t), "status")

	util.MustPollServiceInstanceForCompletion(t, fixtures.ServiceInstanceName, rsp)
}

// TestServiceInstancePollServiceIDOptional tests that the service ID supplied to a service
// instance polling operation is optional.
func TestServiceInstancePollServiceIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	query.Del(util.QueryServiceID)
	util.MustGet(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, query), http.StatusOK, nil)
}

// TestServiceInstancePollPlanIDOptional tests that the plan ID supplied to a service
// instance polling operation is optional.
func TestServiceInstancePollPlanIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	query.Del(util.QueryPlanID)
	util.MustGet(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, query), http.StatusOK, nil)
}

// TestServiceInstancePollIllegalQuery tests polling a completed service instance creation
// with a malformed query results in a bad request error.
func TestServiceInstancePollIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	query := util.PollServiceInstanceQuery(req, rsp)
	util.MustGetAndError(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, query)+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the service ID supplied to a service
// instance polling operation must match that of the instance's service ID.
func TestServiceInstancePollIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	req.ServiceID = fixtures.IllegalID
	util.MustGetAndError(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, util.PollServiceInstanceQuery(req, rsp)), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the plan ID supplied to a service
// instance polling operation must match that of the instance's plan ID.
func TestServiceInstancePollIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.IllegalID
	util.MustGetAndError(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, util.PollServiceInstanceQuery(req, rsp)), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstancePollIllegalServiceID tests that the operation ID supplied to a service
// instance polling operation must match that of the current operation.
func TestServiceInstancePollIllegalOperationID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	rsp := util.MustCreateServiceInstance(t, fixtures.ServiceInstanceName, req)

	rsp.Operation = fixtures.IllegalID
	util.MustGetAndError(t, util.ServiceInstancePollURI(fixtures.ServiceInstanceName, util.PollServiceInstanceQuery(req, rsp)), http.StatusBadRequest, api.ErrorQueryError)
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
	util.MustPut(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, nil)
}

// TestServiceInstanceRecreateAfterCompletion tests the behaviour of multiple creation requests
// for the same service instance with the same request twice, after the operation has
// completed e.g. been acknowledged, should return a 200.
func TestServiceInstanceRecreateAfterCompletion(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPut(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusOK, req, nil)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, api.ErrorResourceConflict)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, api.ErrorResourceConflict)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, api.ErrorResourceConflict)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, api.ErrorResourceConflict)
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
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusConflict, req, api.ErrorResourceConflict)
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

	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, nil), http.StatusUnprocessableEntity, api.ErrorAsyncRequired)
}

// TestServiceInstanceDeleteIllegalInstance tests  service instance deletion when there
// isn't a corresponding service instance.
func TestServiceInstanceDeleteIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusGone, api.ErrorResourceGone)
}

// TestServiceInstanceDeleteIllegalQuery tests that a malformed query raises a bad request
// with a query error.
func TestServiceInstanceDeleteIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery())+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteServiceIDRequired tests delete requests without service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Del(util.QueryServiceID)
	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteServiceIDInvalid tests delete requests with the wrong service_id are
// rejected.
func TestServiceInstanceDeleteServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Set(util.QueryServiceID, fixtures.IllegalID)
	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDRequired tests delete requests without plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Del(util.QueryPlanID)
	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeletePlanIDInvalid tests delete requests with the wrong plan_id are
// rejected.
func TestServiceInstanceDeletePlanIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.DeleteServiceInstanceQuery(req)
	query.Set(util.QueryPlanID, fixtures.IllegalID)
	util.MustDeleteAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceDeleteAndRecreate tests that persistent data isn't left lying about
// and we can recreate an instance.
func TestServiceInstanceDeleteAndRecreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustDeleteServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
	util.MustResetDynamicClient(t, clients)
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestServiceInstanceRead tests that we can read an existing service instance.
func TestServiceInstanceRead(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	read := &api.GetServiceInstanceResponse{}
	util.MustGet(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.ReadServiceInstanceQuery(req)), http.StatusOK, read)

	util.Assert(t, read.ServiceID == req.ServiceID)
	util.Assert(t, read.PlanID == req.PlanID)
}

// TestServiceInstanceReadWithParameters tests that parameters are preserved and
// reported with a get call.
func TestServiceInstanceReadWithParameters(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	read := &api.GetServiceInstanceResponse{}
	util.MustGet(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.ReadServiceInstanceQuery(req)), http.StatusOK, read)

	util.Assert(t, read.ServiceID == req.ServiceID)
	util.Assert(t, read.PlanID == req.PlanID)
	util.Assert(t, reflect.DeepEqual(read.Parameters, req.Parameters))
}

// TestServiceInstanceReadIllegalQuery tests that a malformed query raises a bad request
// with a query error.
func TestServiceInstanceReadIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	util.MustGetAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query)+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadServiceIDOptional tests that we can read an existing service instance
// and the service_id parameter is optional.
func TestServiceInstanceReadServiceIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Del(util.QueryServiceID)
	util.MustGet(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusOK, nil)
}

// TestServiceInstanceReadPlanIDOptional tests that we can read an existing service instance
// and the plan_id parameter is optional.
func TestServiceInstanceReadPlanIDOptional(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Del(util.QueryPlanID)
	util.MustGet(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusOK, nil)
}

// TestServiceInstanceReadServiceIDInvalid tests that we can read an existing service instance
// and the service_id parameter is llegal.
func TestServiceInstanceReadServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Set(util.QueryServiceID, fixtures.IllegalID)
	util.MustGetAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadPlanIDIllegal tests that we can read an existing service instance
// and the plan_id parameter is illegal.
func TestServiceInstanceReadPlanIDIllegal(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	query := util.ReadServiceInstanceQuery(req)
	query.Set(util.QueryPlanID, fixtures.IllegalID)
	util.MustGetAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceInstanceReadIllegalInstance tests that a read on an illegal service
// instance is rejected.
func TestServiceInstanceReadIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustGetAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.ReadServiceInstanceQuery(req)), http.StatusNotFound, api.ErrorResourceNotFound)
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

// TestServiceInstanceUpdateNotAsynchronous tests that update operations must
// be asynchronous.
func TestServiceInstanceUpdateNotAsynchronous(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, nil), http.StatusUnprocessableEntity, nil, api.ErrorAsyncRequired)
}

// TestServiceInstanceUpdateIllegalBody tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, `illegal`, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalQuery tests that an illegal query raises a bad
// request and query error.
func TestServiceInstanceUpdateIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery())+"&%illegal", http.StatusBadRequest, nil, api.ErrorQueryError)
}

// TestServiceInstanceUpdateIllegalServiceID tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalServiceID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.ServiceID = fixtures.IllegalID
	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalPlanID tests that an illegal body raises a bad
// request and parameter error.
func TestServiceInstanceUpdateIllegalPlanID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.IllegalID
	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceUpdatePlanUpdateIllegal tests that updating a service plan when not
// allowed by the catalog responds with a parameter error.
func TestServiceInstanceUpdatePlanUpdateIllegal(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	req.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestServiceInstanceUpdateIllegalInstance tests that an illegal instance raises a not
// found and resource not found error.
func TestServiceInstanceUpdateIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	update := fixtures.BasicServiceInstanceUpdateRequest()
	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusNotFound, update, api.ErrorResourceNotFound)
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
// invalid parameters.
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
	util.MustPatchAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, update, api.ErrorValidationError)
}

// TestServiceInstanceUpdateUpdatedParameters tests that updating a parameter updates
// the underlying resource.
func TestServiceInstanceUpdateUpdatedParameters(t *testing.T) {
	defer mustReset(t)

	optionalParameterValue := "piglet"

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	fixtures.AssertFixtureFieldNotSet(t, clients, "spec", "hostname")

	update := fixtures.BasicServiceInstanceUpdateRequest()
	update.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"` + fixtures.OptionalParameter + `":"` + optionalParameterValue + `"}`),
	}
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)

	fixtures.AssertFixtureFieldSet(t, clients, optionalParameterValue, "spec", "hostname")
}

// TestServiceInstanceUpdatePreserveExternalMutations tests that mutations made by
// Kubernetes are preserved e.g. ports changing could be a problem for someone, it
// shouldn't be, but it will be.
func TestServiceInstanceUpdatePreserveExternalMutations(t *testing.T) {
	defer mustReset(t)

	muatatedValue := "dragon"

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	fixtures.MustSetFixtureField(t, clients, muatatedValue, "spec", "subdomain")

	update := fixtures.BasicServiceInstanceUpdateRequest()
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)

	fixtures.AssertFixtureFieldSet(t, clients, muatatedValue, "spec", "subdomain")
}

// TestServiceInstanceUpdateUpdatedParametersWithExternalMutations tests that updating a parameter
// updates the underlying resource while preserving external updates.
func TestServiceInstanceUpdateUpdatedParametersWithExternalMutations(t *testing.T) {
	defer mustReset(t)

	optionalParameterValue := "chameleon"
	muatatedValue := "parakeet"

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	fixtures.AssertFixtureFieldNotSet(t, clients, "spec", "hostname")
	fixtures.MustSetFixtureField(t, clients, muatatedValue, "spec", "subdomain")

	update := fixtures.BasicServiceInstanceUpdateRequest()
	update.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"` + fixtures.OptionalParameter + `":"` + optionalParameterValue + `"}`),
	}
	util.MustUpdateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, update)

	fixtures.AssertFixtureFieldSet(t, clients, optionalParameterValue, "spec", "hostname")
	fixtures.AssertFixtureFieldSet(t, clients, muatatedValue, "spec", "subdomain")
}
