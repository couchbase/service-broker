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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/test/unit/fixtures"
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

// TestToSnakeCase test conversion of keys to snake case for catalog data
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"already_snake", "already_snake"},
		{"A", "a"},
		{"AA", "aa"},
		{"AaAa", "aa_aa"},
		{"HTTPRequest", "http_request"},
		{"BatteryLifeValue", "battery_life_value"},
		{"Id0Value", "id0_value"},
		{"ID0Value", "id0_value"},
	}
	for _, test := range tests {
		have := broker.ToSnakeCase(test.input)
		if have != test.want {
			t.Errorf("input=%q:\nhave: %q\nwant: %q", test.input, have, test.want)
		}
	}
}

// TestSnakeCaseCatalogKeys test conversion of keys to snake case for catalog data
func TestSnakeCaseCatalogKeys(t *testing.T) {
	catalog := map[string]interface{}{"fooBar": 1, "bar": "2", "foo": true, "barFoo": false}

	broker.SnakeCaseCatalogKeys(catalog, false)

	if _, ok := catalog["fooBar"]; ok {
		t.Errorf("catalog does still contain the key 'fooBar'")
	}

	if _, ok := catalog["barFoo"]; ok {
		t.Errorf("catalog does still contain the key 'barFoo'")
	}

	if _, ok := catalog["foo_bar"]; !ok {
		t.Errorf("catalog does contain converted key 'foo_bar'")
	}

	if _, ok := catalog["bar_foo"]; !ok {
		t.Errorf("catalog does contain converted key 'bar_foo'")
	}
}

// TestSnakeCaseCatalog test conversion of keys to snake case for catalog struct
func TestSnakeCaseCatalog(t *testing.T) {
	catalog := fixtures.BasicConfiguration().Catalog
	catalog.Services[0].PlanUpdatable = true

	catalogInterface, err := broker.SnakeCaseCatalog(catalog)
	if err != nil {
		t.Errorf("failed to snake case catalog: %s", err.Error())
	}

	jsonData, err := json.Marshal(catalogInterface)
	if err != nil {
		t.Errorf("failed to marshal catalog data to json: %s", err.Error())
	}

	var i struct {
		Services []struct {
			PlanUpdatable bool `json:"plan_updatable,omitempty"`
		}
	}

	err = json.Unmarshal(jsonData, &i)
	if err != nil {
		t.Errorf("failed to unmarshal catalog data from json: %s", err.Error())
	}

	if i.Services[0].PlanUpdatable == false {
		t.Errorf("plan_updatable is not set to true")
	}
}
