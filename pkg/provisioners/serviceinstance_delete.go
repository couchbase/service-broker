package provisioners

import (
	"github.com/couchbase/service-broker/pkg/registry"
)

// ServiceInstanceDeleter caches various data associated with deleting a service instance.
type ServiceInstanceDeleter struct {
	// registry is the instance registry.
	registry *registry.Registry

	// instanceID is the instance ID to delete.
	instanceID string
}

// NewServiceInstanceDeleter returns a new controller capable of deleting a service instance.
func NewServiceInstanceDeleter(registry *registry.Registry, instanceID string) *ServiceInstanceDeleter {
	return &ServiceInstanceDeleter{
		registry:   registry,
		instanceID: instanceID,
	}
}

// Run performs asynchronous update tasks.
func (d *ServiceInstanceDeleter) Run() error {
	return d.registry.Delete(registry.ServiceInstanceRegistryName(d.instanceID))
}
