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
	"time"

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

	// namespaceRegistryEntry is the metadata key for accessing the namespace from a template parameter.
	namespaceRegistryEntry = string(registry.Namespace)

	// dashboardURLMutationFormat describes how to turn the input sources to an output value.
	dashboardURLMutationFormat = "http://%v.%v.svc"

	// namespaceRegistryKey is used to lookup the service instance namespace.
	namespaceRegistryKey = "namespace"

	// dashboardURLRegistryKey is the name of the dashboard registry item to set.
	dashboardURLRegistryKey = "dashboard-url"

	// crdentialsRegistryKey is the name of the credentials registry item to set.
	crdentialsRegistryKey = "credentials"

	// instanceNameRegistryEntry is the unique instance name to store in the registry.
	instanceNameRegistryEntry = "instance-name"

	// falseBool is an addressable boolean false.
	falseBool = false

	// zeroInt is an addressable integer zero.
	zeroInt = 0

	// credentialsSnippetName is the name of a credentials snippet.
	credentialsSnippetName = "credentials-snippet"

	// dnsSnippetName is the name of a template snippet.
	dnsSnippetName = "dns-snippet"

	// dnsDefault is an addressable DNS server name.
	dnsDefault = "192.168.0.1"

	// dnsSnippetNamespacePath is an addressable JSON pointer.
	dnsSnippetNamespacePath = "/nameservers/-"

	// basicResourceNamePath is an addressable JSON pointer.
	basicResourceNamePath = "/metadata/name"

	// basicResourceAutomountPath is an addressable JSON pointer.
	basicResourceAutomountPath = "/spec/automountServiceAccountToken"

	// basicResourcePriorityPath is an addressable JSON pointer.
	basicResourcePriorityPath = "/spec/priority"

	// basicResourceContainersPath is an addressable JSON pointer.
	basicResourceContainersPath = "/spec/containers/-"

	// basicResourceDNSPath is an addressable JSON pointer.
	basicResourceDNSPath = "/spec/dnsConfig"

	// optionalParameterPath is used to access the optional parameter.
	optionalParameterPath = "/hostname"

	// optionalParameterPatchPath is used to apply the optional parameter.
	optionalParameterPatchPath = "/spec/hostname"

	// basicResource is used to test object creation, and conflict handling.
	basicResource = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "image",
					Image: "org/image:tag",
				},
			},
		},
	}

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
					Raw: []byte(`{"nameservers":[]}`),
				},
				Parameters: []v1.ConfigurationParameter{
					{
						Name: "nameserver",
						Default: &v1.Literal{
							String: &dnsDefault,
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &dnsSnippetNamespacePath},
						},
					},
				},
			},
			{
				Name: credentialsSnippetName,
				Template: &runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			},
			{
				Name: "test-template",
				// Populated by the configuration function.
				Template: &runtime.RawExtension{},
				Parameters: []v1.ConfigurationParameter{
					{
						Name: "instance-name",
						Source: &v1.ConfigurationParameterSource{
							Accessor: v1.Accessor{
								Registry: &instanceNameRegistryEntry,
							},
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &basicResourceNamePath},
						},
					},
					{
						Name: "automount-service-token",
						Default: &v1.Literal{
							Bool: &falseBool,
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &basicResourceAutomountPath},
						},
					},
					{
						Name: "priority",
						Default: &v1.Literal{
							Int: &zeroInt,
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &basicResourcePriorityPath},
						},
					},
					{
						Name: "sidecar",
						Default: &v1.Literal{
							Object: &runtime.RawExtension{
								Raw: []byte(`{"name":"sidecar","image":"org/sidecar:tag"}`),
							},
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &basicResourceContainersPath},
						},
					},
					{
						Name: "dns",
						Source: &v1.ConfigurationParameterSource{
							Template: &dnsSnippetName,
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &basicResourceDNSPath},
						},
					},
					{
						Name: "hostname",
						Source: &v1.ConfigurationParameterSource{
							Accessor: v1.Accessor{
								Parameter: &optionalParameterPath,
							},
						},
						Destinations: []v1.ConfigurationParameterDestination{
							{Path: &optionalParameterPatchPath},
						},
					},
				},
			},
		},
		Bindings: []v1.ConfigurationBinding{
			{
				Name:    "test-binding",
				Service: "test-offering",
				Plan:    "test-plan",
				ServiceInstance: v1.ServiceBrokerTemplateList{
					Parameters: []v1.ConfigurationParameter{
						{
							Name: "instance-name",
							Source: &v1.ConfigurationParameterSource{
								Format: &v1.ConfigurationParameterSourceFormat{
									String: "instance-%s",
									Parameters: []v1.Accessor{
										{
											Registry: &instanceIDRegistryEntry,
										},
									},
								},
							},
							Destinations: []v1.ConfigurationParameterDestination{
								{Registry: &instanceNameRegistryEntry},
							},
						},
						{
							Name: "dashboard-url",
							Source: &v1.ConfigurationParameterSource{
								Format: &v1.ConfigurationParameterSourceFormat{
									String: dashboardURLMutationFormat,
									Parameters: []v1.Accessor{
										{
											Registry: &instanceNameRegistryEntry,
										},
										{
											Registry: &namespaceRegistryEntry,
										},
									},
								},
							},
							Destinations: []v1.ConfigurationParameterDestination{
								{Registry: &dashboardURLRegistryKey},
							},
						},
					},
					Templates: []string{
						"test-template",
					},
				},
				ServiceBinding: &v1.ServiceBrokerTemplateList{
					Parameters: []v1.ConfigurationParameter{
						{
							Name: "credentials",
							Source: &v1.ConfigurationParameterSource{
								Template: &credentialsSnippetName,
							},
							Destinations: []v1.ConfigurationParameterDestination{
								{Registry: &crdentialsRegistryKey},
							},
						},
					},
				},
			},
			{
				Name:    "test-binding-2",
				Service: "test-offering",
				Plan:    "test-plan-2",
				ServiceInstance: v1.ServiceBrokerTemplateList{
					Parameters: []v1.ConfigurationParameter{
						{
							Name: "instance-name",
							Source: &v1.ConfigurationParameterSource{
								Format: &v1.ConfigurationParameterSourceFormat{
									String: "instance-%s",
									Parameters: []v1.Accessor{
										{
											Registry: &instanceIDRegistryEntry,
										},
									},
								},
							},
							Destinations: []v1.ConfigurationParameterDestination{
								{Registry: &instanceNameRegistryEntry},
							},
						},
					},
				},
				ServiceBinding: &v1.ServiceBrokerTemplateList{
					Parameters: []v1.ConfigurationParameter{
						{
							Name: "credentials",
							Source: &v1.ConfigurationParameterSource{
								Template: &credentialsSnippetName,
							},
							Destinations: []v1.ConfigurationParameterDestination{
								{Registry: &crdentialsRegistryKey},
							},
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
				Namespace: v1.String{
					Accessor: v1.Accessor{
						Registry: &namespaceRegistryKey,
					},
				},
				Name: v1.String{
					Accessor: v1.Accessor{
						Registry: &instanceNameRegistryEntry,
					},
				},
				Type:   "Ready",
				Status: "True",
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
	configuration := basicConfiguration.DeepCopy()

	raw, err := json.Marshal(basicResource)
	if err != nil {
		return nil
	}

	for index, template := range configuration.Templates {
		if template.Name == "test-template" {
			configuration.Templates[index].Template.Raw = raw
		}
	}

	return configuration
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

// RegistryParametersToRegistryWithDefault returns a parameter list as specified.
func RegistryParametersToRegistryWithDefault(key, destination, defaultValue string, required bool) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.ConfigurationParameterSource{
				Accessor: v1.Accessor{
					Registry: &key,
				},
			},
			Default: &v1.Literal{
				String: &defaultValue,
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// ParametersToRegistry returns a parameter list as specified.
func ParametersToRegistry(path, destination string, required bool) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.ConfigurationParameterSource{
				Accessor: v1.Accessor{
					Parameter: &path,
				},
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// DefaultParameterToRegistry return a parameter with a string default only.
func DefaultParameterToRegistry(destination, defaultValue string) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name: "test-parameter",
			Default: &v1.Literal{
				String: &defaultValue,
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// ParametersToRegistryWithDefault returns a parameter list as specified.
func ParametersToRegistryWithDefault(path, destination, defaultValue string, required bool) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.ConfigurationParameterSource{
				Accessor: v1.Accessor{
					Parameter: &path,
				},
			},
			Default: &v1.Literal{
				String: &defaultValue,
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// KeyParameterToRegistry creates a parameter that creates a key of the desired type
// and stores it in the registry.
func KeyParameterToRegistry(t v1.KeyType, e v1.KeyEncodingType, bits *int, destination string) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name: "test-private-key",
			Source: &v1.ConfigurationParameterSource{
				GenerateKey: &v1.ConfigurationParameterSourceGenerateKey{
					Type:     t,
					Encoding: e,
					Bits:     bits,
				},
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// CertificateParameterToRegistry creates a parameter that creates a self-signed ceritificate.
func CertificateParameterToRegistry(key *string, cn string, usage v1.CertificateUsage, destination string) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name: "test-certificate",
			Source: &v1.ConfigurationParameterSource{
				GenerateCertificate: &v1.ConfigurationParameterSourceGenerateCertificate{
					Key: v1.Accessor{
						Registry: key,
					},
					Lifetime: metav1.Duration{
						Duration: time.Hour,
					},
					Usage: usage,
				},
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

// SignedCertificateParameterToRegistry creates a parameter that creates a signed certificate.
func SignedCertificateParameterToRegistry(key *string, cn string, usage v1.CertificateUsage, caKey, caCert *string, destination string) []v1.ConfigurationParameter {
	parameters := CertificateParameterToRegistry(key, cn, usage, destination)
	parameters[0].Source.GenerateCertificate.CA = &v1.SigningCA{
		Key: v1.Accessor{
			Registry: caKey,
		},
		Certificate: v1.Accessor{
			Registry: caCert,
		},
	}

	return parameters
}

// SignedCertificateParameterToRegistryWithDNSSANs creates a parameter that creates a signed certificate.
// This accepts a list of subject alternative names as a string array.  It builds defaulted parameters
// of each and then returns this with the certificate request that consumes them appended.
func SignedCertificateParameterToRegistryWithDNSSANs(key *string, cn string, usage v1.CertificateUsage, sans []string, caKey, caCert *string, destination string) []v1.ConfigurationParameter {
	sanRegistryNames := make([]v1.Accessor, len(sans))
	parameters := []v1.ConfigurationParameter{}

	for index, san := range sans {
		name := fmt.Sprintf("san-%d", index)
		sanRegistryNames[index] = v1.Accessor{
			Registry: &name,
		}

		parameters = append(parameters, DefaultParameterToRegistry(name, san)...)
	}

	certParameters := SignedCertificateParameterToRegistry(key, cn, usage, caKey, caCert, destination)
	certParameters[0].Source.GenerateCertificate.AlternativeNames = &v1.SubjectAlternativeNames{
		DNS: sanRegistryNames,
	}

	return append(parameters, certParameters...)
}

// SignedCertificateParameterToRegistryWitEmailSANs creates a parameter that creates a signed certificate.
// This accepts a list of subject alternative names as a string array.  It builds defaulted parameters
// of each and then returns this with the certificate request that consumes them appended.
func SignedCertificateParameterToRegistryWithEmailSANs(key *string, cn string, usage v1.CertificateUsage, sans []string, caKey, caCert *string, destination string) []v1.ConfigurationParameter {
	sanRegistryNames := make([]v1.Accessor, len(sans))
	parameters := []v1.ConfigurationParameter{}

	for index, san := range sans {
		name := fmt.Sprintf("san-%d", index)
		sanRegistryNames[index] = v1.Accessor{
			Registry: &name,
		}

		parameters = append(parameters, DefaultParameterToRegistry(name, san)...)
	}

	certParameters := SignedCertificateParameterToRegistry(key, cn, usage, caKey, caCert, destination)
	certParameters[0].Source.GenerateCertificate.AlternativeNames = &v1.SubjectAlternativeNames{
		Email: sanRegistryNames,
	}

	return append(parameters, certParameters...)
}

// PasswordParameterToRegistry create a parameter that creates a password of the desired
// length and stores it in the registry.
func PasswordParameterToRegistry(length int, dictionary *string, destination string) []v1.ConfigurationParameter {
	return []v1.ConfigurationParameter{
		{
			Name: "test-password",
			Source: &v1.ConfigurationParameterSource{
				GeneratePassword: &v1.ConfigurationParameterSourceGeneratePassword{
					Length:     length,
					Dictionary: dictionary,
				},
			},
			Destinations: []v1.ConfigurationParameterDestination{
				{Registry: &destination},
			},
		},
	}
}

var (
	fixtureGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
)

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

func AssertFixtureFieldNotSet(t *testing.T, clients client.Clients, path ...string) {
	object, err := clients.Dynamic().Resource(fixtureGVR).Namespace(util.Namespace).Get("instance-"+ServiceInstanceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := unstructured.NestedFieldCopy(object.Object, path...); ok {
		t.Fatal("path found in fixture")
	}
}
