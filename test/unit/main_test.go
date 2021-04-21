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

package unit_test

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
// Tests are then run against them.
func TestMain(m *testing.M) {
	flag.Parse()

	// Load up the test TLS configuration (valid for DNS:localhost).
	cert, err := tls.X509KeyPair([]byte(util.Cert), []byte(util.Key))
	if err != nil {
		fmt.Println("failed to initialize TLS:", err)
		os.Exit(errorCode)
	}

	token := util.Token

	configuration := &broker.ServerConfiguration{
		Namespace:   util.Namespace,
		Token:       &token,
		Certificate: cert,
	}

	// Create fake clients we can use to mock Kubernetes and have complete
	// control over.
	clients, err = util.NewClients()
	if err != nil {
		fmt.Println("failed to initialize clients:", err)
		os.Exit(errorCode)
	}

	// Configure the server.
	if err := broker.ConfigureServer(clients, configuration); err != nil {
		fmt.Println("failed to configure service broker server:", err)
		os.Exit(errorCode)
	}

	// Start the server.
	go func() {
		_ = broker.RunServer(configuration)
	}()

	// Synchronize on server readiness.
	if err := util.WaitFor(util.ServerRunning, time.Minute); err != nil {
		fmt.Println("failed to wait for service broker listening")
	}

	// Run the test suite.
	os.Exit(m.Run())
}
