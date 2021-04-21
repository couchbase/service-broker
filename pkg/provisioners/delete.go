// Copyright 2020-2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
