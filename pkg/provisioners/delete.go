package provisioners

import (
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"
)

// Deleter caches various data associated with deleting a service instance.
type Deleter struct{}

// NewDeleter returns a new controller capable of deleting a service instance.
func NewDeleter() *Deleter {
	return &Deleter{}
}

// Run performs asynchronous update tasks.
func (d *Deleter) Run(entry *registry.Entry) {
	if err := entry.Delete(); err != nil {
		glog.Infof("failed to delete instance")
	}
}
