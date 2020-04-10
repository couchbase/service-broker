package acceptance

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
