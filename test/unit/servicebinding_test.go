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
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/test/unit/fixtures"
	"github.com/couchbase/service-broker/test/unit/util"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestServiceBindingCreate tests service binding creation executes successfully.
func TestServiceBindingCreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}

// TestServiceBindingCreateIllegalBody tests graceful handing of an illegal body.
func TestServiceBindingCreateIllegalBody(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, `illegal`, api.ErrorParameterError)
}

// TestServiceBindingCreateInvalidInstance tests graceful handling of a non-existent service instance.
func TestServiceBindingCreateInvalidInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorParameterError)
}

// TestServiceBindingCreateInvalidService tests graceful handling of a non-existent service ID.
func TestServiceBindingCreateInvalidService(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.ServiceID = fixtures.IllegalID
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorParameterError)
}

// TestServiceBindingCreateInvalidPlan tests graceful handling of a non-existent plan ID.
func TestServiceBindingCreateInvalidPlan(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.PlanID = fixtures.IllegalID
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorParameterError)
}

// TestServiceBindingCreateUnbindableService tests graceful handling of unbindable service offerings.
func TestServiceBindingCreateUnbindableService(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Bindable = false
	configuration.Bindings[0].ServiceBinding = nil
	configuration.Bindings[1].ServiceBinding = nil
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorConfigurationError)
}

// TestServiceBindingCreateUnbindablePlan tests graceful handling of unbindable service plans.
func TestServiceBindingCreateUnbindablePlan(t *testing.T) {
	defer mustReset(t)

	planBindable := false

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Bindable = &planBindable
	configuration.Bindings[0].ServiceBinding = nil
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorConfigurationError)
}

// TestServiceBindingCreateWithSchema tests that schema validation works.
func TestServiceBindingCreateWithSchema(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}

// TestServiceBindingCreateWithSchemaNoParameters tests that schema validation works with no parameters.
func TestServiceBindingCreateWithSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}

// TestServiceBindingCreateWithSchemaInvalid tests that schema validation rejects things that
// don't pass schema validation.
func TestServiceBindingCreateWithSchemaInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchema()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":"string"}`),
	}
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorValidationError)
}

// TestServiceBindingCreateWithRequiredSchema tests that required schemas are handled
// correctly.
func TestServiceBindingCreateWithRequiredSchema(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchemaBindingRequired()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}

// TestServiceBindingCreateWithRequiredSchemaNoParameters tests that required schemas are handled
// correctly.
func TestServiceBindingCreateWithRequiredSchemaNoParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Catalog.Services[0].Plans[0].Schemas = fixtures.BasicSchemaBindingRequired()
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorValidationError)
}

// TestServiceBindingIllegalResource tests that the error handling for a failed service
// instance creation happens gracefully.
func TestServiceBindingIllegalResource(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceBinding.Templates = []string{fixtures.IllegalTemplateName}
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusBadRequest, binding, api.ErrorConfigurationError)
}

// TestServiceBindingRereateAfterCreation tests service binding recreation executes successfully when
// a service binding already exists.
func TestServiceBindingRecreateAfterCreation(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	util.MustPut(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusOK, req, nil)
}

// TestServiceBindingRecreateAfterCreationMismatchedPlanIID tests recreation of a binding
// with a different plan results in a conflict error.
func TestServiceBindingRecreateAfterCreationMismatchedPlanIID(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	binding.PlanID = fixtures.BasicConfigurationPlanID2
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusConflict, binding, api.ErrorResourceConflict)
}

// TestServiceBindingRecreateAfterCreationMismatchedNoContext tests recreation of a binding
// with a context where there was none results in a conflict error.
func TestServiceBindingRecreateAfterCreationMismatchedNoContext(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	binding.Context = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusConflict, binding, api.ErrorResourceConflict)
}

// TestServiceBindingRecreateAfterCreationMismatchedWithContext tests recreation of a binding
// without a context where there was one results in a conflict error.
func TestServiceBindingRecreateAfterCreationMismatchedWithContext(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.Context = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	binding.Context = nil
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusConflict, binding, api.ErrorResourceConflict)
}

// TestServiceBindingRecreateAfterCreationMismatchedNoParameters tests recreation of a binding
// with parameters where there was none results in a conflict error.
func TestServiceBindingRecreateAfterCreationMismatchedNoParameters(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	binding.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusConflict, binding, api.ErrorResourceConflict)
}

// TestServiceBindingRecreateAfterCreationMismatchedWithParameters tests recreation of a binding
// without parameters where there was some results in a conflict error.
func TestServiceBindingRecreateAfterCreationMismatchedWithParameters(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	binding.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"test":1}`),
	}
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	binding.Parameters = nil
	util.MustPutAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, nil), http.StatusConflict, binding, api.ErrorResourceConflict)
}

// TestServiceBindingDelete tests service binding deletion executes successfully.
func TestServiceBindingDelete(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
	util.MustDeleteServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}

// TestServiceBindingDeleteIllegalInstance tests gradeful handling of missing service instances.
func TestServiceBindingDeleteIllegalInstance(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, util.DeleteServiceBindingQuery(binding)), http.StatusBadRequest, api.ErrorParameterError)
}

// TestServiceBindingDeleteIllegalBinding tests graceful handling of missing service bindings.
func TestServiceBindingDeleteIllegalBinding(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, util.DeleteServiceBindingQuery(binding)), http.StatusGone, api.ErrorResourceGone)
}

// TestServiceBindingDeleteIllegalQuery tests graceful handling of malformed queries.
func TestServiceBindingDeleteIllegalQuery(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, util.DeleteServiceBindingQuery(binding))+"&%illegal", http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceBindingDeleteServiceIDRequired tests the service ID parameter is required.
func TestServiceBindingDeleteServiceIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	query := util.DeleteServiceBindingQuery(binding)
	query.Del(util.QueryServiceID)
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceBindingDeleteServiceIDInvalid tests graceful handling of incorrect service IDs.
func TestServiceBindingDeleteServiceIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	query := util.DeleteServiceBindingQuery(binding)
	query.Set(util.QueryServiceID, fixtures.IllegalID)
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceBindingDeletePlanIDRequired tests the plan ID is required.
func TestServiceBindingDeletePlanIDRequired(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	query := util.DeleteServiceBindingQuery(binding)
	query.Del(util.QueryPlanID)
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceBindingDeletePlanIDInvalid tests graceful handling of incorrect plan IDs.
func TestServiceBindingDeletePlanIDInvalid(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())
	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)

	query := util.DeleteServiceBindingQuery(binding)
	query.Set(util.QueryPlanID, fixtures.IllegalID)
	util.MustDeleteAndError(t, util.ServiceBindingURI(fixtures.ServiceInstanceName, fixtures.ServiceBindingName, query), http.StatusBadRequest, api.ErrorQueryError)
}

// TestServiceBindingDeleteDeleteAndRecreate tests that a binding can be recreated after
// it has been deleted.
func TestServiceBindingDeleteDeleteAndRecreate(t *testing.T) {
	defer mustReset(t)

	util.MustReplaceBrokerConfig(t, clients, fixtures.BasicConfiguration())

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	binding := fixtures.BasicServiceBindingCreateRequest()
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
	util.MustDeleteServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
	util.MustCreateServiceBinding(t, fixtures.ServiceInstanceName, fixtures.ServiceBindingName, binding)
}
