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

// TestRegistry tests registry items are correctly populated by service instance
// creation.
func TestRegistry(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.Namespace, util.Namespace)
	util.MustHaveRegistryEntry(t, entry, registry.InstanceID, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.ServiceID, fixtures.BasicConfigurationOfferingID)
	util.MustHaveRegistryEntry(t, entry, registry.PlanID, fixtures.BasicConfigurationPlanID)
	util.MustHaveRegistryEntry(t, entry, registry.DashboardURL, fixtures.DashboardURL)
}

// TestRegistryIllegalWrite tests system registry items are not writable.
func TestRegistryIllegalWrite(t *testing.T) {
	defer mustReset(t)

	illegalKey := string(registry.ServiceID)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters[0].Destination.Registry = &illegalKey
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestRegistryIllegalRead tests that some system registry items are not readable.
func TestRegistryIllegalRead(t *testing.T) {
	defer mustReset(t)

	illegalKey := string(registry.Parameters)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters[0].Source.Registry = &illegalKey
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestRegistryMissingKey tests that referencing a missing key is okay.
func TestRegistryMissingKey(t *testing.T) {
	defer mustReset(t)

	missingKey := "missing"

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters[0].Source.Registry = &missingKey
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, fixtures.ServiceInstanceName)
	util.MustNotHaveRegistryEntry(t, entry, fixtures.DashboardURL)
}

// TestRegistryMissingRequiredKey tests that referencing a missing key is a parameter error.
func TestRegistryMissingRequiredKey(t *testing.T) {
	defer mustReset(t)

	missingKey := "missing"

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters[0].Source.Registry = &missingKey
	configuration.Bindings[0].ServiceInstance.Parameters[0].Required = true
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestRegistryExplicitNamespace tests that the context can update the registry namespace.
func TestRegistryExplicitNamespace(t *testing.T) {
	defer mustReset(t)

	namespace := "BattleCat"

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Context = &runtime.RawExtension{
		Raw: []byte(`{"namespace":"` + namespace + `"}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.Namespace, namespace)
}

// TestRegistryExplicitIllegaNamespace tests that a faulty context raises a parameter
// error
func TestRegistryExplicitIllegaNamespace(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Context = &runtime.RawExtension{
		Raw: []byte(`{"namespace":1}`),
	}
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestRegistryDefault tests that missing registry entries can be defaulted.
func TestRegistryDefault(t *testing.T) {
	defer mustReset(t)

	key := "animal"
	defaultValue := "kitten"

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.RegistryParametersToRegistryWithDefault(key, key, defaultValue, false)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntry(t, entry, registry.Key(key), defaultValue)
}

// TestRegistryiNoDestination test that a configuration error is raised when the destinationisn't specified.
func TestRegistryiNoDestination(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters[0].Destination.Registry = nil
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}
