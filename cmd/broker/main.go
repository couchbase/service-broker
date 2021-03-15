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

// authenticationType is the type of authentication the broker should use.
type authenticationType string

const (
	// bearerToken authentication just does a string match.
	bearerToken authenticationType = "token"

	// basic authentication alos does a string match, or username and password.
	// Note Cloud Foundry expects basic auth.
	basic authenticationType = "basic"
)

// Set sets the authentication type from CLI parameters.
func (a *authenticationType) Set(s string) error {
	switch t := authenticationType(s); t {
	case bearerToken, basic:
		*a = t
	default:
		return fmt.Errorf("%w: unexpected authentication type %s", ErrFatal, s)
	}

	return nil
}

// Type returns the type of flag to display.
func (a *authenticationType) Type() string {
	return "string"
}

// String returns the default authentication type.
func (a *authenticationType) String() string {
	return string(*a)
}

func main() {
	// authenticationType is the type of authentication to use.
	authentication := basic

	// tokenPath is the location of the file containing the bearer token for authentication.
	var tokenPath string

	// usernamePath is the location of the file containing the username for authentication.
	var usernamePath string

	// passwordPath is the location of the file containing the password for authentication.
	var passwordPath string

	// tlsCertificatePath is the location of the file containing the TLS server certifcate.
	var tlsCertificatePath string

	// tlsPrivateKeyPath is the location of the file containing the TLS private key.
	var tlsPrivateKeyPath string

	flag.Var(&authentication, "authentication", "Authentication type to use, either 'basic' or 'token'")
	flag.StringVar(&tokenPath, "token", "/var/run/secrets/service-broker/token", "Bearer token for API authentication")
	flag.StringVar(&usernamePath, "username", "/var/run/secrets/service-broker/username", "Username for basic authentication")
	flag.StringVar(&passwordPath, "password", "/var/run/secrets/service-broker/password", "Password for basic authentication")
	flag.StringVar(&tlsCertificatePath, "tls-certificate", "/var/run/secrets/service-broker/tls-certificate", "Path to the server TLS certificate")
	flag.StringVar(&tlsPrivateKeyPath, "tls-private-key", "/var/run/secrets/service-broker/tls-private-key", "Path to the server TLS key")
	flag.StringVar(&config.ConfigurationName, "config", config.ConfigurationNameDefault, "Configuration resource name")
	flag.Parse()

	// Start the server.
	glog.Infof("%s %s (git commit %s)", version.Application, version.Version, version.GitCommit)

	c := broker.ServerConfiguration{}

	// Parse implicit configuration.
	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		glog.Fatal(fmt.Errorf("%w: NAMESPACE environment variable must be set", ErrFatal))
		os.Exit(errorCode)
	}

	c.Namespace = namespace

	// Load up explicit configuration.
	switch authentication {
	case bearerToken:
		token, err := ioutil.ReadFile(tokenPath)
		if err != nil {
			glog.Fatal(err)
			os.Exit(errorCode)
		}

		stringToken := string(token)
		c.Token = &stringToken

	case basic:
		username, err := ioutil.ReadFile(usernamePath)
		if err != nil {
			glog.Fatal(err)
			os.Exit(errorCode)
		}

		password, err := ioutil.ReadFile(passwordPath)
		if err != nil {
			glog.Fatal(err)
			os.Exit(errorCode)
		}

		stringUsername := string(username)
		stringPassword := string(password)

		c.BasicAuth = &broker.ServerConfigurationBasicAuth{
			Username: stringUsername,
			Password: stringPassword,
		}
	}

	cert, err := tls.LoadX509KeyPair(tlsCertificatePath, tlsPrivateKeyPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	c.Certificate = cert

	// Initialize the clients.
	clients, err := client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	// Start the server.
	if err := broker.ConfigureServer(clients, &c); err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}

	if err := broker.RunServer(&c); err != nil {
		glog.Fatal(err)
		os.Exit(errorCode)
	}
}
