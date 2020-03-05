package provisioners

import (
	"github.com/couchbase/service-broker/pkg/registry"
)

// ServiceBindingCreator caches various data associated with deleting a service instance.
type ServiceBindingCreator struct {
	// registry is the instance registry.
	registry *registry.Registry

	// instanceID is the instance ID to create.
	instanceID string

	// bindingID is the binding ID to create.
	bindingID string
}

// NewServiceBindingCreator returns a new controller capable of deleting a service instance.
func NewServiceBindingCreator(registry *registry.Registry, instanceID, bindingID string) *ServiceBindingCreator {
	return &ServiceBindingCreator{
		registry:   registry,
		instanceID: instanceID,
		bindingID:  bindingID,
	}
}

// Run performs asynchronous update tasks.
func (d *ServiceBindingCreator) Run() error {
	return nil
}
