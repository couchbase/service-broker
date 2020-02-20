package test

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"

	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/test/util"
)

// TestMain creates, initializes and starts the service broker locally.
// Tests are then run against the
func TestMain(m *testing.M) {
	cert, err := tls.X509KeyPair([]byte(util.Cert), []byte(util.Key))
	if err != nil {
		fmt.Println("Failed to initialize TLS:", err)
		os.Exit(1)
	}

	clients, err := util.NewClients()
	if err != nil {
		fmt.Println("Failed to initialize clients:", err)
		os.Exit(1)
	}

	if err := broker.ConfigureServer(clients, util.Namespace, util.Token); err != nil {
		fmt.Println("Failed to configure service broker server:", err)
		os.Exit(1)
	}

	go func() {
		_ = broker.RunServer(cert)
	}()

	os.Exit(m.Run())
}
