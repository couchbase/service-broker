package config

import (
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type configuration struct {
	// clients is the set of clients this instance of the broker uses, by default
	// this will use in-cluster Kubernetes, however may be replaced by fake clients
	// by a test framework.
	clients client.Clients

	// config is the user supplied configuration custom resource.
	config *v1.CouchbaseServiceBrokerConfig

	// token is the API access token.
	token string

	// namespace is the default namespace the broker is running in.
	namespace string
}

// c is the global configuration struct.
var c *configuration

// Configure initializes global configuration and must be called before starting
// the API service.
func Configure(clients client.Clients, namespace, token string) error {
	brokerConfig, err := clients.Broker().BrokerV1().CouchbaseServiceBrokerConfigs(namespace).Get("couchbase-service-broker", metav1.GetOptions{})
	if err != nil {
		return err
	}

	c = &configuration{
		clients:   clients,
		config:    brokerConfig,
		token:     token,
		namespace: namespace,
	}

	return nil
}

// Clients returns a set of Kubernetes clients.
func Clients() client.Clients {
	return c.clients
}

// Config returns the user specified custom resource.
func Config() *v1.CouchbaseServiceBrokerConfig {
	return c.config
}

// Token returns the API bearer token.
func Token() string {
	return c.token
}

// Namespace returns the broker namespace.
func Namespace() string {
	return c.namespace
}
