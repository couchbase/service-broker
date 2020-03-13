package fixtures

import (
	"encoding/json"

	"github.com/couchbase/service-broker/pkg/api"
	v1 "github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1alpha1"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ServiceInstanceName is a name to use for a service instance.
	ServiceInstanceName = "pinkiepie"

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
)

var (
	// instanceIDRegistryEntry is the metadata key for accessing the instance ID from a template parameter.
	instanceIDRegistryEntry = string(registry.InstanceID)

	// namespaceRegistryEntry is the metadata key for accessing the namespace from a template parameter.
	namespaceRegistryEntry = string(registry.Namespace)

	// dashboardURLMutationFormat describes how to turn the input sources to an output value.
	dashboardURLMutationFormat = "http://%v.%v.svc"

	// dashboardURLRegistryKey is the name of the dashboard registry item to set.
	dashboardURLRegistryKey = "dashboard-url"

	// instanceNameRegistryEntry is the unique instance name to store in the registry.
	instanceNameRegistryEntry = "instance-name"

	// falseBool is an addressable boolean false.
	falseBool = false

	// zeroInt is an addressable integer zero.
	zeroInt = 0

	// dnsSnippetName is the name of a template snippet.
	dnsSnippetName = "dns-snippet"

	// dnsDefault is an addressable DNS server name.
	dnsDefault = "192.168.0.1"

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

	// basicConfiguration is the absolute minimum valid configuration allowed by the
	// service broker configuration schema.
	basicConfiguration = &v1.CouchbaseServiceBrokerConfigSpec{
		Catalog: &v1.ServiceCatalog{
			Services: []v1.ServiceOffering{
				{
					Name:        "test-offering",
					ID:          BasicConfigurationOfferingID,
					Description: "a test offering",
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
		Templates: []v1.CouchbaseServiceBrokerConfigTemplate{
			{
				Name: dnsSnippetName,
				Template: &runtime.RawExtension{
					Raw: []byte(`{"nameservers":[]}`),
				},
				Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameter{
					{
						Name: "nameserver",
						Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
							String: &dnsDefault,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/nameservers/-",
							},
						},
					},
				},
			},
			{
				Name: "test-template",
				// Populated by the configuration function.
				Template: &runtime.RawExtension{},
				Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameter{
					{
						Name: "instance-name",
						Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
							Registry: &instanceNameRegistryEntry,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/metadata/name",
							},
						},
					},
					{
						Name: "automount-service-token",
						Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
							Bool: &falseBool,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/spec/automountServiceAccountToken",
							},
						},
					},
					{
						Name: "priority",
						Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
							Int: &zeroInt,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/spec/priority",
							},
						},
					},
					{
						Name: "sidecar",
						Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
							Object: &runtime.RawExtension{
								Raw: []byte(`{"name":"sidecar","image":"org/sidecar:tag"}`),
							},
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/spec/containers/-",
							},
						},
					},
					{
						Name: "dns",
						Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
							Template: &dnsSnippetName,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Paths: []string{
								"/spec/dnsConfig",
							},
						},
					},
				},
			},
		},
		Bindings: []v1.CouchbaseServiceBrokerConfigBinding{
			{
				Name:    "test-binding",
				Service: "test-offering",
				Plan:    "test-plan",
				ServiceInstance: &v1.CouchbaseServiceBrokerTemplateList{
					Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameter{
						{
							Name: "instance-name",
							Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
								Format: &v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormat{
									String: "instance-%s",
									Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter{
										{
											Registry: &instanceIDRegistryEntry,
										},
									},
								},
							},
							Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
								Registry: &instanceNameRegistryEntry,
							},
						},
						{
							Name: "dashboard-url",
							Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
								Format: &v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormat{
									String: dashboardURLMutationFormat,
									Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter{
										{
											Registry: &instanceNameRegistryEntry,
										},
										{
											Registry: &namespaceRegistryEntry,
										},
									},
								},
							},
							Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
								Registry: &dashboardURLRegistryKey,
							},
						},
					},
					Templates: []string{
						"test-template",
					},
				},
			},
			{
				Name:    "test-binding-2",
				Service: "test-offering",
				Plan:    "test-plan-2",
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
)

// EmptyConfiguration returns an empty configuration, useful for testing when users
// really screw up.
func EmptyConfiguration() *v1.CouchbaseServiceBrokerConfigSpec {
	return &v1.CouchbaseServiceBrokerConfigSpec{}
}

// BasicConfiguration is the absolute minimum valid configuration allowed by the
// service broker configuration schema.
func BasicConfiguration() *v1.CouchbaseServiceBrokerConfigSpec {
	configuration := basicConfiguration.DeepCopy()

	raw, err := json.Marshal(basicResource)
	if err != nil {
		return nil
	}

	configuration.Templates[1].Template.Raw = raw

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

// RegistryParametersToRegistryWithDefault returns a parameter list as specified.
func RegistryParametersToRegistryWithDefault(key, destination, defaultValue string, required bool) []v1.CouchbaseServiceBrokerConfigTemplateParameter {
	return []v1.CouchbaseServiceBrokerConfigTemplateParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
				Registry: &key,
			},
			Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
				String: &defaultValue,
			},
			Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
				Registry: &destination,
			},
		},
	}
}

// ParametersToRegistry returns a parameter list as specified.
func ParametersToRegistry(path, destination string, required bool) []v1.CouchbaseServiceBrokerConfigTemplateParameter {
	return []v1.CouchbaseServiceBrokerConfigTemplateParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
				Parameter: &path,
			},
			Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
				Registry: &destination,
			},
		},
	}
}

// ParametersToRegistryWithDefault returns a parameter list as specified.
func ParametersToRegistryWithDefault(path, destination, defaultValue string, required bool) []v1.CouchbaseServiceBrokerConfigTemplateParameter {
	return []v1.CouchbaseServiceBrokerConfigTemplateParameter{
		{
			Name:     "test-parameter",
			Required: required,
			Source: &v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
				Parameter: &path,
			},
			Default: &v1.CouchbaseServiceBrokerConfigTemplateParameterDefault{
				String: &defaultValue,
			},
			Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
				Registry: &destination,
			},
		},
	}
}
