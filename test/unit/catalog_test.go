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

package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/test/unit/util"
)

var (
	// testCatalogUpdateFixture is a catalog fixture used to test that
	// the catalog can be updated.
	testCatalogUpdateFixture = &v1.ServiceCatalog{
		Services: []v1.ServiceOffering{
			{
				Name: "fluttershy",
			},
		},
	}
)

// TestCatalogUpdate tests that catalog updates are reflected in the API.
func TestCatalogUpdate(t *testing.T) {
	defer mustReset(t)

	callback := func(config *v1.ServiceBrokerConfig) {
		config.Spec.Catalog = *testCatalogUpdateFixture
	}
	util.MustUpdateBrokerConfig(t, clients, callback)

	validator := func() error {
		catalog := &util.ServiceCatalog{}
		if err := util.Get("/v2/catalog", http.StatusOK, catalog); err != nil {
			return err
		}

		if len(catalog.Services) != 1 || catalog.Services[0].Name != "fluttershy" {
			return fmt.Errorf("catalog not updated as expected")
		}

		return nil
	}
	util.MustWaitFor(t, validator, time.Minute)
}
