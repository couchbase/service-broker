package main

import (
	"crypto/tls"
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
		glog.Fatal(fmt.Errorf("NAMESPACE environment variable must be set"))
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
