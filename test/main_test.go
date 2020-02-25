package test

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/test/util"
)

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
	clients, err := util.NewClients()
	if err != nil {
		fmt.Println("failed to initialize clients:", err)
		os.Exit(1)
	}

	// Configure the server.
	if err := broker.ConfigureServer(clients, util.Namespace, util.Token); err != nil {
		fmt.Println("failed to configure service broker server:", err)
		os.Exit(1)
	}

	// Synchronize on server readiness.
	if err := util.WaitFor(config.Ready, time.Minute); err != nil {
		fmt.Println("failed to wait for service broker ready condition")
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
