package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/test/util"
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
	callback := func(config *v1.CouchbaseServiceBrokerConfig) {
		config.Spec.Catalog = testCatalogUpdateFixture
	}
	util.MustUpdateBrokerConfig(t, clients, callback)

	validator := func() bool {
		catalog := &util.ServiceCatalog{}
		if err := util.Get("/v2/catalog", http.StatusOK, catalog); err != nil {
			return false
		}
		if len(catalog.Services) != 1 || catalog.Services[0].Name != "fluttershy" {
			return false
		}
		return true
	}
	util.MustWaitFor(t, validator, time.Minute)
}
