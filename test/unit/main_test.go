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
	"github.com/couchbase/service-broker/test/unit/util"
)

const (
	// errorCode is what to return on application error.
	errorCode = 1
)

var (
	// clients are a global set of clients for test cases to use.
	// They are initialzed so that the server will start, and can be
	// written to to trigger behviours, witness consequences and
	// verify actions.  They should be reset after each test.
	clients client.Clients
)

// reset cleans the client of any resources that we may have registered and
// clears out any broker persistent state.
func reset() error {
	return util.ResetClients(clients)
}

// mustResetcleans the client of any resources that we may have registered and
// clears out any broker persistent state.
func mustReset(t *testing.T) {
	if err := reset(); err != nil {
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
		os.Exit(errorCode)
	}

	// Create fake clients we can use to mock Kubernetes and have complete
	// control over.
	clients, err = util.NewClients()
	if err != nil {
		fmt.Println("failed to initialize clients:", err)
		os.Exit(errorCode)
	}

	// Configure the server.
	if err := broker.ConfigureServer(clients, util.Namespace, util.Token); err != nil {
		fmt.Println("failed to configure service broker server:", err)
		os.Exit(errorCode)
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
