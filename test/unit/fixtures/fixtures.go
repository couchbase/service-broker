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

package fixtures

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/unit/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// ServiceInstanceName is a name to use for a service instance.
	ServiceInstanceName = "pinkiepie"

	// AlternateServiceInstanceName is a name to use for another service instance.
	AlternateServiceInstanceName = "rainbowdash"

	// ServiceBindingName is a name to use for a service binding.
	ServiceBindingName = "spike"

	// IllegalID is an illegal ID and must not be used as a service or plan ID.
	IllegalID = "illegal"

	// BasicConfigurationOfferingID is the symbolic constant UUID used for the basic configuration.
	BasicConfigurationOfferingID = `dd2cce49-a0ff-4deb-9cbf-b97301fdb87e`

	// BasicConfigurationPlanID is the symbolic constant UUID used for the basic configuration plan.
	BasicConfigurationPlanID = `3f525c60-bd66-4b91-8d18-beba57fbc0b8`

	// BasicConfigurationPlanID2 is the symbolic constant UUID used for the basic configuration plan.
	BasicConfigurationPlanID2 = `e18fce8d-1f4a-44fa-88c8-e7a52ed50f29`

	// BasicSchemaParameters is a simple schema for use in parameter validation.
	BasicSchemaParameters = `{"$schema":"http://json-schema.org/draft-04/schema#","type":"object","properties":{"test":{"type":"number","minimum":1}}}`

	// BasicSchemaParametersRequired is a simple schema for use in parameter validation.
	BasicSchemaParametersRequired = `{"$schema":"http://json-schema.org/draft-04/schema#","type":"object","required":["test"],"properties":{"test":{"type":"number","minimum":1}}}`

	// DashboardURL is the expected dashboard URL to be generated.
	DashboardURL = "http://instance-" + ServiceInstanceName + "." + util.Namespace + ".svc"

	// OptionalParameter is a template parameter this is optional.
	OptionalParameter = "hostname"
)

var (
	// instanceIDRegistryEntry is the metadata key for accessing the instance ID from a template parameter.
	instanceIDRegistryEntry = string(registry.InstanceID)

	// credentialsSnippetName is the name of a credentials snippet.
	credentialsSnippetName = "credentials-snippet"

	// dnsSnippetName is the name of a template snippet.
	dnsSnippetName = "dns-snippet"

	// dnsDefault is an addressable DNS server name.
	dnsDefault = "192.168.0.1"

	// basicResourceStatus allow the resource to pass the readiness checks
	// defined for it.
	basicResourceStatus = &corev1.PodStatus{
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	// basicConfiguration is the absolute minimum valid configuration allowed by the
	// service broker configuration schema.
	basicConfiguration = &v1.ServiceBrokerConfigSpec{
		Catalog: v1.ServiceCatalog{
			Services: []v1.ServiceOffering{
				{
					Name:        "test-offering",
					ID:          BasicConfigurationOfferingID,
					Description: "a test offering",
					Bindable:    true,
					Plans: []v1.ServicePlan{
						{
							Name:        "test-plan",
							ID:          BasicConfigurationPlanID,
							Description: "a test plan",
						},
						{
							Name:        "test-plan-2",
							ID:          BasicConfigurationPlanID2,
							Description: "another test plan",
						},
					},
				},
			},
		},
		Templates: []v1.ConfigurationTemplate{
			{
				Name: dnsSnippetName,
				Template: &runtime.RawExtension{
					Raw: []byte(`{"nameservers":["{{ default \"` + dnsDefault + `\" nil }}"]}`),
				},
			},
			{
				Name: credentialsSnippetName,
				Template: &runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			},
			{
				Name:     "test-template",
				Template: &runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"{{ registry \"instance-name\" }}"},"spec":{"containers":[{"name":"image","image":"name/image:tag"}],"automountServiceAccountToken":"{{ true }}","priority":"{{ 0 }}","dnsConfig":"{{ snippet \"dns-snippet\" }}","hostname":"{{ parameter \"/hostname\" }}"}}`)},
			},
			{
				Name:      "test-singleton",
				Singleton: true,
				Template:  &runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"singleton"}}`)},
			},
		},
		Bindings: []v1.ConfigurationBinding{
			{
				Name:    "test-binding",
				Service: "test-offering",
				Plan:    "test-plan",
				ServiceInstance: v1.ServiceBrokerTemplateList{
					Registry: []v1.RegistryValue{
						{
							Name:  "instance-name",
							Value: "{{ printf \"instance-%s\" (registry \"instance-id\") }}",
						},
						{
							Name:  "dashboard-url",
							Value: "{{ printf \"http://%v.%v.svc\" (registry \"instance-name\") (registry \"namespace\") }}",
						},
					},
					Templates: []string{
						"test-template",
						"test-singleton",
					},
				},
				ServiceBinding: &v1.ServiceBrokerTemplateList{
					Registry: []v1.RegistryValue{
						{
							Name:  "credentials",
							Value: "{{ snippet \"" + credentialsSnippetName + "\" }}",
						},
					},
				},
			},
			{
				Name:    "test-binding-2",
				Service: "test-offering",
				Plan:    "test-plan-2",
				ServiceInstance: v1.ServiceBrokerTemplateList{
					Registry: []v1.RegistryValue{
						{
							Name:  "instance-name",
							Value: "{{ registry \"" + instanceIDRegistryEntry + "\" }}",
						},
					},
				},
				ServiceBinding: &v1.ServiceBrokerTemplateList{
					Registry: []v1.RegistryValue{
						{
							Name:  "credentials",
							Value: "{{ snippet \"" + credentialsSnippetName + "\" }}",
						},
					},
				},
			},
		},
	}

	// basicReadinessChecks are used to check the test resource created by our
	// service instance is ready.
	basicReadinessChecks = v1.ConfigurationReadinessCheckList{
		{
			Name: "pod-ready",
			Condition: &v1.ConfigurationReadinessCheckCondition{
				APIVersion: "v1",
				Kind:       "Pod",
				Namespace:  `{{ registry "namespace" }}`,
				Name:       `{{ registry "instance-name" }}`,
				Type:       "Ready",
				Status:     "True",
			},
		},
	}

	// basicSchema is schema for service instance validation with optional parameters.
	basicSchema = &v1.Schemas{
		ServiceInstance: &v1.ServiceInstanceSchema{
			Create: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParameters),
				},
			},
			Update: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParameters),
				},
			},
		},
		ServiceBinding: &v1.ServiceBindingSchema{
			Create: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParameters),
				},
			},
		},
	}

	// basicSchemaRequired is a schema for service instance validation with required parameters.
	basicSchemaRequired = &v1.Schemas{
		ServiceInstance: &v1.ServiceInstanceSchema{
			Create: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParametersRequired),
				},
			},
			Update: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParametersRequired),
				},
			},
		},
	}

	// basicSchemaBindingRequired is a schema for a service binding with required parameters.
	basicSchemaBindingRequired = &v1.Schemas{
		ServiceBinding: &v1.ServiceBindingSchema{
			Create: &v1.InputParamtersSchema{
				Parameters: &runtime.RawExtension{
					Raw: []byte(BasicSchemaParametersRequired),
				},
			},
		},
	}

	// basicServiceInstanceCreateRequest is the absolute minimum valid service instance create
	// request to use against the basicConfiguration.
	basicServiceInstanceCreateRequest = api.CreateServiceInstanceRequest{
		ServiceID: BasicConfigurationOfferingID,
		PlanID:    BasicConfigurationPlanID,
	}

	// basicServiceInstanceUpdateRequest is the absolute minimum valid service instance update
	// request to use against the basicConfiguration.
	basicServiceInstanceUpdateRequest = api.UpdateServiceInstanceRequest{
		ServiceID: BasicConfigurationOfferingID,
	}

	// basicServiceBindingCreateRequest is the absolute minimum valid service bindinf create
	// request to use against the basicConfiguration.
	basicServiceBindingCreateRequest = api.CreateServiceBindingRequest{
		ServiceID: BasicConfigurationOfferingID,
		PlanID:    BasicConfigurationPlanID,
	}
)

// EmptyConfiguration returns an empty configuration, useful for testing when users
// really screw up.
func EmptyConfiguration() *v1.ServiceBrokerConfigSpec {
	return &v1.ServiceBrokerConfigSpec{}
}

// BasicConfiguration is the absolute minimum valid configuration allowed by the
// service broker configuration schema.
func BasicConfiguration() *v1.ServiceBrokerConfigSpec {
	return basicConfiguration.DeepCopy()
}

// BasicConfigurationWithReadiness returns the standard configuration with a readiness
// check added for the resource that is templated.
func BasicConfigurationWithReadiness() *v1.ServiceBrokerConfigSpec {
	configuration := BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.ReadinessChecks = basicReadinessChecks.DeepCopy()

	return configuration
}

// BasicSchema is schema for service instance create validation with optional parameters.
func BasicSchema() *v1.Schemas {
	return basicSchema.DeepCopy()
}

// BasicSchemaRequired is a schema for service instance create validation with required parameters.
func BasicSchemaRequired() *v1.Schemas {
	return basicSchemaRequired.DeepCopy()
}

// BasicSchemaBindingRequired is a schema for service binding create validation with required parameters.
func BasicSchemaBindingRequired() *v1.Schemas {
	return basicSchemaBindingRequired.DeepCopy()
}

// BasicServiceInstanceCreateRequest is the absolute minimum valid service instance create
// request to use against the basicConfiguration.
func BasicServiceInstanceCreateRequest() *api.CreateServiceInstanceRequest {
	return basicServiceInstanceCreateRequest.DeepCopy()
}

// BasicServiceInstanceUpdateRequest is the absolute minimum valid service instance update
// request to use against the basicConfiguration.
func BasicServiceInstanceUpdateRequest() *api.UpdateServiceInstanceRequest {
	return basicServiceInstanceUpdateRequest.DeepCopy()
}

// BasicServiceBindingCreateRequest is the absolute minimum valid service bindinf create
// request to use against the basicConfiguration.
func BasicServiceBindingCreateRequest() *api.CreateServiceBindingRequest {
	return basicServiceBindingCreateRequest.DeepCopy()
}

// BasicResourceStatus returns the templated resource status that will fulfil the
// readiness checks defined above.  The unstructured application code needs things to
// be fully unstructured, it's not clever enough to use a raw object type.
func BasicResourceStatus(t *testing.T) interface{} {
	raw, err := json.Marshal(basicResourceStatus)
	if err != nil {
		t.Fatal(err)
	}

	var object interface{}
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}

	return object
}

// SetRegistry sets the service instance binding registry entries for the first binding
// to the requested template expression.
func SetRegistry(spec *v1.ServiceBrokerConfigSpec, name string, expression interface{}) {
	var str string

	switch t := expression.(type) {
	case Function:
		str = string(t)
	case Pipeline:
		str = string(t)
	case string, int, bool, nil:
		str = argument(t)
	default:
		fmt.Println("fail")
	}

	spec.Bindings[0].ServiceInstance.Registry = []v1.RegistryValue{
		{
			Name:  name,
			Value: `{{` + str + `}}`,
		},
	}
}

// AddRegistry appends the requested template expression to the service instance binding
// registry entry for the first binding.
func AddRegistry(spec *v1.ServiceBrokerConfigSpec, name string, expression interface{}) {
	var str string

	switch t := expression.(type) {
	case Function:
		str = string(t)
	case Pipeline:
		str = string(t)
	case string, int, bool, nil:
		str = argument(t)
	default:
		fmt.Println("fail")
	}

	spec.Bindings[0].ServiceInstance.Registry = append(spec.Bindings[0].ServiceInstance.Registry, v1.RegistryValue{
		Name:  name,
		Value: `{{` + str + `}}`,
	})
}

var (
	fixtureGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
)

// MustSetFixtureField sets the named field in the fixture Kubernetes resource.
func MustSetFixtureField(t *testing.T, clients client.Clients, value interface{}, path ...string) {
	object, err := clients.Dynamic().Resource(fixtureGVR).Namespace(util.Namespace).Get("instance-"+ServiceInstanceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if err := unstructured.SetNestedField(object.Object, value, path...); err != nil {
		t.Fatal(err)
	}

	if _, err := clients.Dynamic().Resource(fixtureGVR).Namespace(util.Namespace).Update(object, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
}

// AssertFixtureFieldSet asserts that the named field in the Kubernetes resource is
// set as expected.
func AssertFixtureFieldSet(t *testing.T, clients client.Clients, value interface{}, path ...string) {
	object, err := clients.Dynamic().Resource(fixtureGVR).Namespace(util.Namespace).Get("instance-"+ServiceInstanceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	objValue, ok, _ := unstructured.NestedFieldCopy(object.Object, path...)
	if !ok {
		t.Fatal("path not found in fixture")
	}

	if !reflect.DeepEqual(objValue, value) {
		t.Fatal("value mismatch", objValue, value)
	}
}

// AssertFixtureFieldNotSet asserts that the named field in the Kubernetes resource
// is not set as expected.
func AssertFixtureFieldNotSet(t *testing.T, clients client.Clients, path ...string) {
	object, err := clients.Dynamic().Resource(fixtureGVR).Namespace(util.Namespace).Get("instance-"+ServiceInstanceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := unstructured.NestedFieldCopy(object.Object, path...); ok {
		t.Fatal("path found in fixture")
	}
}
