package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/fixtures"
	"github.com/couchbase/service-broker/test/util"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// key is the name of the registry key we will create.
	key = "animal"

	// value is the value of the registry key we will create.
	value = "cat"

	// defaultValue is the default value for the registry key to use.
	defaultValue = "kitten"
)

// TestParameters tests parameter items are correctly populated by service instance
// creation.
func TestParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, false)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"` + key + `":"` + value + `"}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.Key(key), value)
}

// TestParametersMissingPath tests parameter items are correctly populated by service instance
// creation.
func TestParametersMissingPath(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, false)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustNotHaveRegistryEntry(t, entry, registry.Key(key))
}

// TestParametersMissingRequiredPath tests parameter items are correctly populated by service instance
// creation.
func TestParametersMissingRequiredPath(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, true)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestParametersDefault tests a parameter with a default work when not specified.
func TestParametersDefault(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistryWithDefault("/animal", key, defaultValue, false)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.Key(key), defaultValue)
}
