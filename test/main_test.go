package test

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/test/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// clients are a global set of clients for test cases to use.
	// They are initialzed so that the server will start, and can be
	// written to to trigger behviours, witness consequences and
	// verify actions.  They should be reset after each test.
	clients client.Clients
)

// resetClients cleans the client of any resources that we may have registered.
// NOTE: we could just blindly replace the client's fake object cache but
// then we'd not trigger any reaction chains e.g. watches.  So play it safe
// and do it manually.
func resetClients() error {
	resources, err := clients.Kubernetes().Discovery().ServerResources()
	if err != nil {
		return err
	}

	deleteOptions := metav1.NewDeleteOptions(0)
	listOptions := metav1.ListOptions{}
	for _, resourceList := range resources {
		for _, resource := range resourceList.APIResources {
			gvr := schema.GroupVersionResource{
				Group:    resource.Group,
				Version:  resource.Version,
				Resource: resource.Name,
			}
			if err := clients.Dynamic().Resource(gvr).DeleteCollection(deleteOptions, listOptions); err != nil {
				return err
			}
		}
	}
	return nil
}

// mustResetClients cleans the client of any resources that we may have registered.
func mustResetClients(t *testing.T) {
	if err := resetClients(); err != nil {
		t.Fatal(err)
	}
}

// TestMain creates, initializes and starts the service broker locally.
// Tests are then run against the
func TestMain(m *testing.M) {
	flag.Parse()

	// Load up the test TLS configuration (valid for DNS:localhost).
	cert, err := tls.X509KeyPair([]byte(util.Cert), []byte(util.Key))
	if err != nil {
		fmt.Println("failed to initialize TLS:", err)
		os.Exit(1)
	}

	// Create fake clients we can use to mock Kubernetes and have complete
	// controll over.
	clients, err = util.NewClients()
	if err != nil {
		fmt.Println("failed to initialize clients:", err)
		os.Exit(1)
	}

	// Configure the server.
	if err := broker.ConfigureServer(clients, util.Namespace, util.Token); err != nil {
		fmt.Println("failed to configure service broker server:", err)
		os.Exit(1)
	}

	// Start the server.
	go func() {
		_ = broker.RunServer(cert)
	}()

	// Synchronize on server readiness.
	if err := util.WaitFor(util.ServerRunning, time.Minute); err != nil {
		fmt.Println("failed to wait for service broker listening")
	}

	// Run the test suite.
	os.Exit(m.Run())
}
