package fixtures

import (
	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/registry"

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
)

var (
	// instanceIDMetadataName is the metadata key for accessing the instance ID from a template parameter.
	instanceIDMetadataName = string(registry.InstanceID)

	// dashboardURLMutationFormat describes how to turn the input sources to an output value.
	dashboardURLMutationFormat = "http://instance-%v.example.com"

	// dashboardURLRegistryKey is the name of the dashboard registry item to set.
	dashboardURLRegistryKey = "dashboard-url"

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
				Name: "test-template",
				Parameters: []v1.CouchbaseServiceBrokerConfigTemplateParameter{
					{
						Name: "dashboard-url",
						Sources: []v1.CouchbaseServiceBrokerConfigTemplateParameterSource{
							{
								Metadata: &instanceIDMetadataName,
							},
						},
						Mutation: &v1.CouchbaseServiceBrokerConfigTemplateParameterMutation{
							Format: &dashboardURLMutationFormat,
						},
						Destination: v1.CouchbaseServiceBrokerConfigTemplateParameterDestination{
							Registry: &dashboardURLRegistryKey,
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
	return basicConfiguration.DeepCopy()
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
