package provisioners

// ResourceType defines the type of resource being operated on.
type ResourceType string

const (
	// ResourceTypeServiceInstance is used to configure service instances.
	ResourceTypeServiceInstance ResourceType = "service-instance"

	// ResourceTypeServiceBinding is used to confugure service bindings.
	ResourceTypeServiceBinding ResourceType = "service-binding"
)
