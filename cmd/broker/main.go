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

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/version"

	"github.com/golang/glog"
)

const (
	// errorCode is what to return on application error.
	errorCode = 1
)

// ErrFatal is raised when the broker is unable to start.
var ErrFatal = errors.New("fatal error")

func main() {
	// tokenPath is the location of the file containing the bearer token for authentication.
	var tokenPath string

	// tlsCertificatePath is the location of the file containing the TLS server certifcate.
	var tlsCertificatePath string

	// tlsPrivateKeyPath is the location of the file containing the TLS private key.
	var tlsPrivateKeyPath string

	flag.StringVar(&tokenPath, "token", "/var/run/secrets/service-broker/token", "Bearer token for API authentication")
	flag.StringVar(&tlsCertificatePath, "tls-certificate", "/var/run/secrets/service-broker/tls-certificate", "Path to the server TLS certificate")
	flag.StringVar(&tlsPrivateKeyPath, "tls-private-key", "/var/run/secrets/service-broker/tls-private-key", "Path to the server TLS key")
	flag.StringVar(&config.ConfigurationName, "config", config.ConfigurationNameDefault, "Configuration resource name")
	flag.Parse()

	// Start the server.
	glog.Infof("%s %s (git commit %s)", version.Application, version.Version, version.GitCommit)

	// Parse implicit configuration.
	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		glog.Fatal(fmt.Errorf("%w: NAMESPACE environment variable must be set", ErrFatal))
		os.Exit(errorCode)
	}

	// Load up explicit configuration.
	token, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	cert, err := tls.LoadX509KeyPair(tlsCertificatePath, tlsPrivateKeyPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	// Initialize the clients.
	clients, err := client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	// Start the server.
	if err := broker.ConfigureServer(clients, namespace, string(token)); err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	if err := broker.RunServer(cert); err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}
}
