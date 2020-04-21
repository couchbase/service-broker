// Copyright 2020 Couchbase, Inc.
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

package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/unit/fixtures"
	"github.com/couchbase/service-broker/test/unit/util"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestRegistry tests registry items are correctly populated by service instance
// creation.
func TestRegistry(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Namespace, util.Namespace)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.InstanceID, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.ServiceID, fixtures.BasicConfigurationOfferingID)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.PlanID, fixtures.BasicConfigurationPlanID)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.DashboardURL, fixtures.DashboardURL)
}

// TestRegistryIllegalWrite tests system registry items are not writable.
func TestRegistryIllegalWrite(t *testing.T) {
	defer mustReset(t)

	illegalKey := string(registry.ServiceID)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, illegalKey, fixtures.NewRegistryPipeline(key).WithDefault(defaultValue))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestRegistryIllegalRead tests that some system registry items are not readable.
func TestRegistryIllegalRead(t *testing.T) {
	defer mustReset(t)

	illegalKey := string(registry.Parameters)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewRegistryPipeline(illegalKey))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestRegistryMissingKey tests that referencing a missing key is okay.
func TestRegistryMissingKey(t *testing.T) {
	defer mustReset(t)

	missingKey := "missing"

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, fixtures.DashboardURL, fixtures.NewRegistryPipeline(missingKey))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustNotHaveRegistryEntry(t, entry, fixtures.DashboardURL)
}

// TestRegistryMissingRequiredKey tests that referencing a missing key is a parameter error.
func TestRegistryMissingRequiredKey(t *testing.T) {
	defer mustReset(t)

	missingKey := "missing"

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewRegistryPipeline(missingKey).Required())
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
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

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Namespace, namespace)
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
	fixtures.SetRegistry(configuration, key, fixtures.NewRegistryPipeline(key).WithDefault(defaultValue))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Key(key), defaultValue)
}
