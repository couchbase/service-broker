package provisioners

import (
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"
)

// ServiceInstanceDeleter caches various data associated with deleting a service instance.
type ServiceInstanceDeleter struct {
	// registry is the instance registry.
	registry *registry.Entry

	// instanceID is the instance ID to delete.
	instanceID string
}

// NewServiceInstanceDeleter returns a new controller capable of deleting a service instance.
func NewServiceInstanceDeleter(registry *registry.Entry, instanceID string) *ServiceInstanceDeleter {
	return &ServiceInstanceDeleter{
		registry:   registry,
		instanceID: instanceID,
	}
}

// Run performs asynchronous update tasks.
func (d *ServiceInstanceDeleter) Run() {
	if err := d.registry.Delete(); err != nil {
		glog.Infof("failed to delete instance")
	}
}
