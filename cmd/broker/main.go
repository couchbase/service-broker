package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/couchbase/service-broker/pkg/broker"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/version"

	"github.com/golang/glog"
)

func main() {
	// Start the server.
	glog.Infof("%s v%s (git commit %s)", version.Application, version.Version, version.GitCommit)

	// Runtime configuration.
	var tokenPath string
	var tlsCertificatePath string
	var tlsPrivateKeyPath string

	flag.StringVar(&tokenPath, "token", "/var/run/secrets/couchbase.com/service-broker/token", "Bearer token for API authentication")
	flag.StringVar(&tlsCertificatePath, "tls-certificate", "/var/run/secrets/couchbase.com/service-broker/tls-certificate", "Path to the server TLS certificate")
	flag.StringVar(&tlsPrivateKeyPath, "tls-private-key", "/var/run/secrets/couchbase.com/service-broker/tls-private-key", "Path to the server TLS key")
	flag.Parse()

	// Parse implicit configuration.
	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		glog.Fatal(fmt.Errorf("NAMESPACE environment variable must be set"))
		os.Exit(1)
	}

	// Load up explicit configuration.
	token, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
	cert, err := tls.LoadX509KeyPair(tlsCertificatePath, tlsPrivateKeyPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Initialize the clients.
	clients, err := client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Start the server.
	if err := broker.ConfigureServer(clients, namespace, string(token)); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
	if err := broker.RunServer(cert); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
}
