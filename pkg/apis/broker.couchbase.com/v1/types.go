package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CouchbaseServiceBrokerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CouchbaseServiceBrokerConfigSpec `json:"spec"`
}

// CouchbaseServiceBrokerConfigSpec defines the top level service broker configuration
// data structure.
type CouchbaseServiceBrokerConfigSpec struct {
	Catalog   *ServiceCatalog                        `json:"catalog,omitempty"`
	Templates []CouchbaseServiceBrokerConfigTemplate `json:"templates,omitempty"`
	Bindings  []CouchbaseServiceBrokerConfigBinding  `json:"bindings,omitempty"`
}

// ServiceCatalog is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceCatalog struct {
	Services []ServiceOffering `json:"services"`
}

// ServiceOffering is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceOffering struct {
	Name                 string                `json:"name,omitempty"`
	ID                   string                `json:"id,omitempty"`
	Description          string                `json:"description,omitempty"`
	Tags                 []string              `json:"tags,omitempty"`
	Requires             []string              `json:"requires,omitempty"`
	Bindable             bool                  `json:"bindable,omitempty"`
	InstancesRetrievable bool                  `json:"instances_retrievable,omitempty"`
	BindingsRetrievable  bool                  `json:"bindings_retrievable,omitempty"`
	AllowContextUpdates  bool                  `json:"allow_context_updates,omitempty"`
	Metadata             *runtime.RawExtension `json:"metadata,omitempty"`
	DashboardClient      *DashboardClient      `json:"dashboard_client,omitempty"`
	PlanUpdatable        bool                  `json:"plan_updatable,omitempty"`
	Plans                []ServicePlan         `json:"plans,omitempty"`
}

// DashboardClient is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type DashboardClient struct {
	ID            string `json:"id"`
	Secret        string `json:"secret"`
	RedirectedURI string `json:"redirected_uri"`
}

// ServicePlan is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServicePlan struct {
	ID                     string            `json:"id,omitempty"`
	Name                   string            `json:"name,omitempty"`
	Description            string            `json:"description,omitempty"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	Free                   bool              `json:"free,omitempty"`
	Bindable               bool              `json:"bindable,omitempty"`
	PlanUpdatable          bool              `json:"plan_updatable,omitempty"`
	Schemas                *Schemas          `json:"schemas,omitempty"`
	MaximumPollingDuration int               `json:"maximum_polling_duration,omitempty"`
	MaintenanceInfo        *MaintenanceInfo  `json:"maintentance_info,omitempty"`
}

// Schemas is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type Schemas struct {
	ServiceInstance *ServiceInstanceSchema `json:"service_instance,omitempty"`
	ServiceBinding  *ServiceBindingSchema  `json:"service_binding,omitempty"`
}

// ServiceInstanceSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceInstanceSchema struct {
	Create *InputParamtersSchema `json:"create,omitempty"`
	Update *InputParamtersSchema `json:"update,omitempty"`
}

// ServiceBindingSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceBindingSchema struct {
	Create *InputParamtersSchema `json:"create,omitempty"`
}

// InputParamtersSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type InputParamtersSchema struct {
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// MaintenanceInfo is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type MaintenanceInfo struct {
	Version string `json:"version,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplate defines a resource template for use when either
// creating a service instance or service binding.
type CouchbaseServiceBrokerConfigTemplate struct {
	// Name is the name of the template
	Name string `json:"name,omitempty"`

	// Template defines the resource template, it can be any kind of resource
	// supported by client-go or couchbase.
	Template *runtime.RawExtension `json:"template,omitempty"`

	// Parameters allow parameters to be sourced either from request metadata
	// or request parameters as defined in the service catalog.  If specified
	// they will override existing values.  If not then the existing config
	// will be left in place.  When there is no existing configuration and no
	// parameter is specified in the request then an optional default value is
	// used.
	Parameters []CouchbaseServiceBrokerConfigTemplateParameter `json:"parameters,omitempty"`

	// Singleton alters the behaviour of resource creation.  Typically we will
	// create a resource and use parameters to alter it's name, ensuring it
	// doesn't already exist.  Singleton resources will first check to see
	// whether they exist before attempting creation.
	Singleton bool `json:"singleton,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameter defines a parameter substitution
// on a resource template.
type CouchbaseServiceBrokerConfigTemplateParameter struct {
	// Name is a textual name used to uniquely identify the parameter for
	// the template.
	Name string `json:"name,omitempty"`

	// Source is source of the parameter, either from request metadata or
	// the request parameters from the client.
	Source CouchbaseServiceBrokerConfigTemplateParameterSource `json:"source,omitempty"`

	// Destination is the destination of the parameter.
	Destination CouchbaseServiceBrokerConfigTemplateParameterDestination `json:"destination,omitempty"`

	// Required will cause an error if the parameter is not specified.
	Required bool `json:"required,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSource defines where parameters
// are sourced from.
type CouchbaseServiceBrokerConfigTemplateParameterSource struct {
	// Metadata, if set, uses the corresponding metadata value for the
	// parameter source.
	Metadata *string `json:"metadata,omitempty"`

	// Parameter, if set, uses the corresponding request parameter for the
	// parameter source.
	Parameter *CouchbaseServiceBrokerConfigTemplateParameterSourceParameter `json:"parameter,omitempty"`

	// Prefix attaches a prefix to string based parameters. Useful for constaining
	// names and the like to beginning with a character e.g DNS hostnames.
	Prefix *string `json:"prefix,omitempty"`

	// Suffix attaches a suffix to string based parameters. Useful for appending
	// a DNS domain for example.
	Suffix *string `json:"suffix,omitempty"`

	// Format allows the parameter to be inserted into a string with a call
	// to fmt.Sprintf.
	Format *string `json:"format,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSourceParameter defines a source
// parameter originating with a request.
type CouchbaseServiceBrokerConfigTemplateParameterSourceParameter struct {
	// Path specifies the path in JSON pointer format to extract
	// the parameter from a request parameter.
	Path string `json:"path,omitempty"`

	// Default specifies the default value to use if a source parameter
	// is not defined in the request parameters.
	Default *runtime.RawExtension `json:"default,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterDestination defines where to
// patch parameters into the resource template.
type CouchbaseServiceBrokerConfigTemplateParameterDestination struct {
	// Paths is a list of JSON pointers in the resource template to patch
	// the parameter.  The service broker will create any parent objects
	// necessary to fulfill the request.
	Paths []string `json:"paths,omitempty"`
}

// CouchbaseServiceBrokerConfigBinding binds a service plan to a set of templates
// required to realize that plan.
type CouchbaseServiceBrokerConfigBinding struct {
	// Name is a unique identifier for the binding.
	Name string `json:"name,omitempty"`

	// Service is the name of the service offering to bind to.
	Service string `json:"service,omitempty"`

	// Plan is the name of the service plan to bind to.
	Plan string `json:"plan,omitempty"`

	// ServiceInstance defines the set of templates to render and create when
	// a new service instance is created.
	ServiceInstance *CouchbaseServiceBrokerTemplateList `json:"serviceInstance,omitempty"`

	// ServiceBinding defines the set of templates to render and create when
	// a new service binding is created.  This attribute is optional based on
	// whether the service plan allows binding.
	ServiceBinding *CouchbaseServiceBrokerTemplateList `json:"serviceBinding,omitempty"`
}

// CouchbaseServiceBrokerTemplateList is an ordered list of templates to use
// when performing a specific operation.
type CouchbaseServiceBrokerTemplateList struct {
	// Templates defines all the templates that will be created, in order,
	// by the service broker for this operation.
	Templates []string `json:"templates,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CouchbaseServiceBrokerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CouchbaseServiceBrokerConfig `json:"items"`
}
