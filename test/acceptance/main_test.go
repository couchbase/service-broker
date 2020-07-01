// Copyright 2020 Couchbase, Inc.
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

package acceptance_test

import (
	"flag"
	"os"
	"testing"

	"github.com/couchbase/service-broker/pkg/apis"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/test/acceptance/util"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// clients is the global cache of clients.
	clients client.Clients
)

// TestMain performs any cluster initialization.
func TestMain(m *testing.M) {
	// For the benefit of glog.
	flag.Parse()

	// Add any custom resource types to the global scheme.
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	var err error

	// Create any clients we need.
	clients, err = client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Delete and recreate any CRDs so we have the most up to date
	// versions installed.
	if err := util.SetupCRDs(clients); err != nil {
		glog.Fatal(err)
		os.Exit(0)
	}

	// Run the tests.
	os.Exit(m.Run())
}
