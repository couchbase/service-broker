package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/test/fixtures"
	"github.com/couchbase/service-broker/test/util"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestServiceInstanceCreateNotAynchronous tests that the service broker rejects service
// instance creation that isn't asynchronous.
func TestServiceInstanceCreateNotAynchronous(t *testing.T) {
	defer mustResetClients(t)

	util.MustPut(t, "/v2/service_instances/pinkiepie", http.StatusUnprocessableEntity, ``)
}

// TestServiceInstanceCreateIllegalBody tests that the service broker rejects service
// instance creation when the body isn't JSON.
func TestServiceInstanceCreateIllegalBody(t *testing.T) {
	defer mustResetClients(t)

	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, `illegal`)
}

// TestServiceInstanceCreateIllegalConfiguration tests that the service broker handles
// misconfiguration of the service catalog gracefully.  On this occasion the default
// doesn't have any service offerings or plans defined.
func TestServiceInstanceCreateIllegalConfiguration(t *testing.T) {
	defer mustResetClients(t)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req)
}

// TestServiceInstanceCreateInvalidService tests that the service broker handles
// an invalid service gracefully.
func TestServiceInstanceCreateInvalidService(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.ServiceID = "illegal"
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req)
}

// TestServiceInstanceCreateInvalidPlan tests that the service broker handles
// an invalid plan gracefully.
func TestServiceInstanceCreateInvalidPlan(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.PlanID = "illegal"
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req)
}

// TestServiceInstanceCreate tests that the service broker accepts a minimal
// service instance creation.
func TestServiceInstanceCreate(t *testing.T) {
	defer mustResetClients(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req)
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
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req)
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
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req)
}

// TestServiceInstanceCreateWithSchemaNoParameters tests that the service broker accepts a
// minimal service instance creation with schema validation and no parameters.
func TestServiceInstanceCreateWithSchemaNoParameters(t *testing.T) {
	defer mustResetClients(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusAccepted, req)
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
	util.MustPut(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req)
}
