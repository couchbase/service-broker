package provisioners

import (
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"
)

// Deleter caches various data associated with deleting a service instance.
type Deleter struct {
	// registry is the instance registry.
	registry *registry.Entry

	// instanceID is the instance ID to delete.
	instanceID string
}

// NewDeleter returns a new controller capable of deleting a service instance.
func NewDeleter(registry *registry.Entry, instanceID string) *Deleter {
	return &Deleter{
		registry:   registry,
		instanceID: instanceID,
	}
}

// Run performs asynchronous update tasks.
func (d *Deleter) Run() {
	if err := d.registry.Delete(); err != nil {
		glog.Infof("failed to delete instance")
	}
}
