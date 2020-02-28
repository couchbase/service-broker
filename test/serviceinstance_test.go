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
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	util.MustPutWithError(t, "/v2/service_instances/pinkiepie", ``, http.StatusUnprocessableEntity, api.ErrorAsyncRequired)
}

// TestServiceInstanceCreateIllegalBody tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalBody(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", `illegal`, http.StatusBadRequest, api.ErrorParameterErrorEXT)
}

// TestServiceInstanceCreateIllegalConfiguration tests that the service broker handles
// misconfiguration of the service catalog gracefully.  On this occasion the default
// doesn't have any service offerings or plans defined.
func TestServiceInstanceCreateIllegalConfiguration(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.EmptyConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorParameterErrorEXT)
}

// TestServiceInstanceCreateIllegalQuery tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalQuery(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true&%illegal", req, http.StatusBadRequest, api.ErrorQueryErrorEXT)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = "illegal"
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorParameterErrorEXT)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = "illegal"
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorParameterErrorEXT)
}

// TestServiceInstanceCreate tests that the service broker accepts a minimal
// service instance creation.
func TestServiceInstanceCreate(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
}

// TestServiceInstanceCreateWithSchema tests that the service broker accepts a
// minimal service instance creation with schema validation.
func TestServiceInstanceCreateWithSchema(t *testing.T) {
	defer mustResetClients(t)

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
	defer mustResetClients(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusAccepted)
}

// TestServiceInstanceCreateSchemaValidationFail tests that the service broker rejects
// schema validation failure.
func TestServiceInstanceCreateSchemaValidationFail(t *testing.T) {
	defer mustResetClients(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":"string"}`),
	}
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorValidationErrorEXT)
}

// TestServiceInstanceCreateWithRequiredSchemaNoParameters tests that the service broker
// rejects a minimal service instance creation with required schema validation and no
// parameters.
func TestServiceInstanceCreateWithRequiredSchemaNoParameters(t *testing.T) {
	defer mustResetClients(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchemaRequired()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutWithError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", req, http.StatusBadRequest, api.ErrorValidationErrorEXT)
}
