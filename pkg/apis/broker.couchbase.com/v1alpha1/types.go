package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=all;couchbase
// +kubebuilder:resource:scope=Namespaced
type CouchbaseServiceBrokerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CouchbaseServiceBrokerConfigSpec `json:"spec"`
}

// CouchbaseServiceBrokerConfigSpec defines the top level service broker configuration
// data structure.
type CouchbaseServiceBrokerConfigSpec struct {
	// Catalog is the service catalog definition and is required.
	Catalog *ServiceCatalog `json:"catalog"`

	// Templates is a set of resource templates that can be rendered by the service broker and is required.
	// +kubebuilder:validation:MinItems=1
	Templates []CouchbaseServiceBrokerConfigTemplate `json:"templates"`

	// Bindings is a set of bindings that link service plans to resource templates and is required.
	// +kubebuilder:validation:MinItems=1
	Bindings []CouchbaseServiceBrokerConfigBinding `json:"bindings"`
}

// ServiceCatalog is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceCatalog struct {
	// Services is an array of Service Offering objects
	Services []ServiceOffering `json:"services"`
}

// ServiceOffering is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceOffering struct {
	// Name is the name of the Service Offering. MUST be unique across all Service Offering
	// objects returned in this response. MUST be a non-empty string. Using a CLI-friendly name
	// is RECOMMENDED.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// ID is an identifier used to correlate this Service Offering in future requests to the
	// Service Broker. This MUST be globally unique such that Platforms (and their users) MUST
	// be able to assume that seeing the same value (no matter what Service Broker uses it) will
	// always refer to this Service Offering. MUST be a non-empty string. Using a GUID is RECOMMENDED.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`

	// Descriptions is a short description of the service. MUST be a non-empty string.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`

	// Tags provide a flexible mechanism to expose a classification, attribute, or base
	// technology of a service, enabling equivalent services to be swapped out without changes
	// to dependent logic in applications, buildpacks, or other services. E.g. mysql, relational,
	// redis, key-value, caching, messaging, amqp.
	Tags []string `json:"tags,omitempty"`

	// Requires is a list of permissions that the user would have to give the service, if they provision
	// it. The only permissions currently supported are syslog_drain, route_forwarding and volume_mount.
	// +kubebuilder:validation:Enum=syslog_drain;route_forwarding;volume_mount
	Requires []string `json:"requires,omitempty"`

	// Bindable specifies whether Service Instances of the service can be bound to applications. This
	// specifies the default for all Service Plans of this Service Offering. Service Plans can override
	// this field (see Service Plan Object).
	Bindable bool `json:"bindable"`

	// InstancesRetrievable specifies whether the Fetching a Service Instance endpoint is supported for
	// all Service Plans.
	InstancesRetrievable bool `json:"instances_retrievable,omitempty"`

	// BindingsRetrievable specifies whether the Fetching a Service Binding endpoint is supported for all
	// Service Plans.
	BindingsRetrievable bool `json:"bindings_retrievable,omitempty"`

	// AllowContextUpdates specifies whether a Service Instance supports Update requests when contextual
	// data for the Service Instance in the Platform changes.
	AllowContextUpdates bool `json:"allow_context_updates,omitempty"`

	// Metadata is an opaque object of metadata for a Service Offering. It is expected that Platforms will
	// treat this as a blob. Note that there are conventions in existing Service Brokers and Platforms for
	// fields that aid in the display of catalog data.
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// Dashboard is a Cloud Foundry extension described in Catalog Extensions. Contains the data necessary
	// to activate the Dashboard SSO feature for this service.
	DashboardClient *DashboardClient `json:"dashboard_client,omitempty"`

	// PlanUpdatable is whether the Service Offering supports upgrade/downgrade for Service Plans by default.
	// Service Plans can override this field (see Service Plan). Please note that the misspelling of the
	// attribute plan_updatable as plan_updateable was done by mistake. We have opted to keep that misspelling
	// instead of fixing it and thus breaking backward compatibility. Defaults to false.
	PlanUpdatable bool `json:"plan_updatable,omitempty"`

	// ServicePlan is a list of Service Plans for this Service Offering, schema is defined below. MUST
	// contain at least one Service Plan.
	// +kubebuilder:validation:MinItems=1
	Plans []ServicePlan `json:"plans"`
}

// DashboardClient is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type DashboardClient struct {
	// ID is the id of the OAuth client that the dashboard will use. If present, MUST be a non-empty string.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`

	// Secret is a secret for the dashboard client. If present, MUST be a non-empty string.
	// +kubebuilder:validation:MinLength=1
	Secret string `json:"secret"`

	// RedirectedURI is a URI for the service dashboard. Validated by the OAuth token server when the dashboard
	// requests a token.
	RedirectedURI string `json:"redirected_uri,omitempty"`
}

// ServicePlan is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServicePlan struct {
	// ID is an identifier used to correlate this Service Offering in future requests to the
	// Service Broker. This MUST be globally unique such that Platforms (and their users) MUST
	// be able to assume that seeing the same value (no matter what Service Broker uses it) will
	// always refer to this Service Offering. MUST be a non-empty string. Using a GUID is RECOMMENDED.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`

	// Name is the name of the Service Plan. MUST be unique within the Service Offering. MUST be
	// a non-empty string. Using a CLI-friendly name is RECOMMENDED.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is a short description of the Service Plan. MUST be a non-empty string.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`

	// Metadata is an opaque object of metadata for a Service Plan. It is expected that Platforms
	// will treat this as a blob. Note that there are conventions in existing Service Brokers and
	// Platforms for fields that aid in the display of catalog data.
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// Free, when false, Service Instances of this Service Plan have a cost. The default is true.
	Free bool `json:"free,omitempty"`

	// Bindable specifies whether Service Instances of the Service Plan can be bound to applications.
	// This field is OPTIONAL. If specified, this takes precedence over the bindable attribute of
	// the Service Offering. If not specified, the default is derived from the Service Offering.
	Bindable *bool `json:"bindable,omitempty"`

	// PlanUpdatable specifies whether the Plan supports upgrade/downgrade/sidegrade to another
	// version. This field is OPTIONAL. If specificed, this takes precedence over the Service
	// Offering's plan_updateable field. If not specified, the default is derived from the Service
	// Offering. Please note that the attribute is intentionally misspelled as plan_updateable
	// for legacy reasons.
	PlanUpdatable *bool `json:"plan_updatable,omitempty"`

	// Schemas are schema definitions for Service Instances and Service Bindings for the Service
	// Plan.
	Schemas *Schemas `json:"schemas,omitempty"`

	// MaximumPollingDuration is a duration, in seconds, that the Platform SHOULD use as the
	// Service's maximum polling duration.
	MaximumPollingDuration int `json:"maximum_polling_duration,omitempty"`

	// MaintenanceInfo is maintenance information for a Service Instance which is provisioned using
	// the Service Plan. If provided, a version string MUST be provided and platforms MAY use this
	// when Provisioning or Updating a Service Instance.
	MaintenanceInfo *MaintenanceInfo `json:"maintentance_info,omitempty"`
}

// Schemas is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type Schemas struct {
	// ServiceInstance is the schema definitions for creating and updating a Service Instance.
	ServiceInstance *ServiceInstanceSchema `json:"service_instance,omitempty"`

	// ServiceBinding is the schema definition for creating a Service Binding. Used only if the
	// Service Plan is bindable.
	ServiceBinding *ServiceBindingSchema `json:"service_binding,omitempty"`
}

// ServiceInstanceSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceInstanceSchema struct {
	// Create is the schema definition for creating a Service Instance.
	Create *InputParamtersSchema `json:"create,omitempty"`

	// Update is the chema definition for updating a Service Instance.
	Update *InputParamtersSchema `json:"update,omitempty"`
}

// ServiceBindingSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type ServiceBindingSchema struct {
	// Create is the schema definition for creating a Service Binding.
	Create *InputParamtersSchema `json:"create,omitempty"`
}

// InputParamtersSchema is defined by:
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#body
type InputParamtersSchema struct {
	// Parameters is the schema definition for the input parameters. Each input parameter is
	// expressed as a property within a JSON object.
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
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Template defines the resource template, it can be any kind of resource
	// supported by client-go or couchbase.
	Template *runtime.RawExtension `json:"template"`

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
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Required will cause an error if a parameter is not defined.
	Required bool `json:"required,omitempty"`

	// Default specifies the default value is if the parameter is not defined.
	Default *CouchbaseServiceBrokerConfigTemplateParameterDefault `json:"default,omitempty"`

	// Source is source of the parameter.
	Source *CouchbaseServiceBrokerConfigTemplateParameterSource `json:"source,omitempty"`

	// Destinations is the destination of the parameter.
	// +kubebuilder:validation:MinItems=1
	Destinations []CouchbaseServiceBrokerConfigTemplateParameterDestination `json:"destinations"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSource defines where parameters
// are sourced from.
type CouchbaseServiceBrokerConfigTemplateParameterSource struct {
	// Registry, if set, uses the corresponding registry value for the
	// parameter source.
	// +kubebuilder:validation:Pattern="^(instance:)?[a-zA-Z0-9-]+$"
	Registry *string `json:"registry,omitempty"`

	// Parameter, if set, uses the corresponding request parameter for the
	// parameter source.
	Parameter *string `json:"parameter,omitempty"`

	// Format allows the collection of an arbitrary number of parameters into
	// a string format.
	Format *CouchbaseServiceBrokerConfigTemplateParameterSourceFormat `json:"format,omitempty"`

	// GeneratePassword allows the generation of a random string, useful for password
	// generation.
	GeneratePassword *CouchbaseServiceBrokerConfigTemplateParameterSourceGeneratePassword `json:"generatePassword,omitempty"`

	// Template allows the recursive rendering and inclusion of a named template.
	Template *string `json:"template,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSourceFormat defines a formatting
// string and parameters.
type CouchbaseServiceBrokerConfigTemplateParameterSourceFormat struct {
	// String is the format string to use.
	String string `json:"string"`

	// Parameters is the set of parameters corresponding to the format string.
	// All parameters must exist or the formatting operation will return nil.
	// +kubebuilder:validation:MinItems=1
	Parameters []CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter `json:"parameters"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSourceGeneratePassword defines a random string.
type CouchbaseServiceBrokerConfigTemplateParameterSourceGeneratePassword struct {
	// Length is the length of the string to generate.
	// +kubebuilder:validation:Minimum=1
	Length int `json:"length"`

	// Dictionary is the string of symbols to use.  This defaults to [a-zA-Z0-9].
	Dictionary *string `json:"dictionary,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter is a parameter
// for a formatting operation.
type CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter struct {
	// Registry , if set, uses the corresponding registry value for the
	// parameter source.
	// +kubebuilder:validation:Pattern="^(instance:)?[a-zA-Z0-9-]+$"
	Registry *string `json:"registry,omitempty"`

	// Parameter, if set, uses the corresponding request parameter for the
	// parameter source.
	Parameter *string `json:"parameter,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterDefault defines a
// default value for a parameter source if it is not specified.
type CouchbaseServiceBrokerConfigTemplateParameterDefault struct {
	// String specifies the default string value if the parameter is not defined.
	String *string `json:"string,omitempty"`

	// Bool specifies the default boolean value if the parameter is not defined.
	Bool *bool `json:"bool,omitempty"`

	// Int specifies the default int value if the parameter is not defined.
	Int *int `json:"int,omitempty"`

	// Object specifies the default value if the parameter is not defined.
	Object *runtime.RawExtension `json:"object,omitempty"`
}

// CouchbaseServiceBrokerConfigTemplateParameterDestination defines where to
// patch parameters into the resource template.
type CouchbaseServiceBrokerConfigTemplateParameterDestination struct {
	// Path is a JSON pointer in the resource template to patch
	// the parameter.
	Path *string `json:"path,omitempty"`

	// Registry is a key to store the value to in the registry.
	Registry *string `json:"registry,omitempty"`
}

// CouchbaseServiceBrokerConfigBinding binds a service plan to a set of templates
// required to realize that plan.
type CouchbaseServiceBrokerConfigBinding struct {
	// Name is a unique identifier for the binding.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Service is the name of the service offering to bind to.
	// +kubebuilder:validation:MinLength=1
	Service string `json:"service"`

	// Plan is the name of the service plan to bind to.
	// +kubebuilder:validation:MinLength=1
	Plan string `json:"plan"`

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
	// Parameters allows registry parameters to be mutated and cached when a
	// service instance is created.  These are only executed on instance creation.
	Parameters []CouchbaseServiceBrokerConfigTemplateParameter `json:"parameters,omitempty"`

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
